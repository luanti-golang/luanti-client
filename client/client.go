// Package client is a headless Luanti client: it connects to a server, logs in
// (optionally registering the account), joins the game and then lets you
// observe every packet and send your own. With Config.SkipJoin it stops before
// joining, after the definitions and media have arrived.
//
//	c := client.New(client.Config{
//		Address:  "localhost:30000",
//		Username: "bot",
//		Password: "secret",
//		Register: true,
//	})
//	client.Handle(c, func(m *protocol.ToClientChatMessage) { fmt.Println(m.Message) })
//	if err := c.Connect(ctx); err != nil { ... }
//	c.Chat("hello world")
//	err := c.Wait()
//
// Packets are plain structs from package luanti-client/protocol.
package client

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/luanti-golang/luanti-client/mapdump"
	"github.com/luanti-golang/luanti-client/protocol"
	"github.com/luanti-golang/luanti-client/srp"
	"github.com/luanti-golang/luanti-client/transport"
	"github.com/luanti-golang/luanti-client/viewerdef"
)

// Config configures a Client. Only Address and Username are required.
type Config struct {
	// Address of the server as host:port, e.g. "localhost:30000".
	Address  string
	Username string
	Password string
	// Register creates the account if the server does not know the name yet.
	// Without it, connecting with an unknown name fails with ErrNotRegistered.
	Register bool

	// Tap, if set, receives every packet in both directions that passes Filter.
	// It is called synchronously and must not block for long.
	Tap func(Event)
	// Filter restricts what is passed to Tap; nil passes everything.
	Filter *Filter
	// OnDatagram, if set, receives every raw UDP datagram (unfiltered).
	OnDatagram func(dir transport.Direction, datagram []byte)

	// Language is sent to the server in TOSERVER_INIT2 (default "en").
	Language string
	// ClientVersion is reported to the server (default "5.18.0-go").
	ClientVersion string
	// RequestMedia downloads all media files announced by the server. Off by
	// default: a headless client rarely needs textures and sounds.
	RequestMedia bool
	// MediaDir, if set, saves downloaded media files into this directory
	// (created if needed) and implies RequestMedia. Files are verified against
	// the SHA-1 the server announced, and files already present with the right
	// hash are not downloaded again.
	MediaDir string
	// NodeDefsFile, if set, receives the server's node definitions as indented
	// JSON as soon as they arrive (see protocol.NodeDefinition for the schema).
	NodeDefsFile string
	// NodeDefsFormat selects the schema of NodeDefsFile: "protocol" (default,
	// see WriteNodeDefsJSON) or "viewer" (see WriteViewerNodeDefsJSON).
	NodeDefsFormat string
	// MapblockDir, if set, saves every received map block into this directory
	// (created if needed) as "(x,y,z).json" in the mapblock format of
	// luanti-viewer, see package mapdump. A block sent again replaces its file.
	MapblockDir string
	// Logf receives warnings (e.g. rejected media files). Default: discarded.
	Logf func(format string, args ...any)
	// DisableBlockAck stops acknowledging received map blocks. Servers stop
	// sending blocks once too many are unacknowledged.
	DisableBlockAck bool
	// ViewRange is the wanted view range in map blocks (16 nodes each) reported
	// to the server. 0 asks for as little map data as possible.
	ViewRange uint8
	// PositionInterval is how often the player position is sent (default 1s).
	PositionInterval time.Duration
	// ConnectTimeout bounds Connect (default 20s). With RequestMedia this
	// includes downloading all media, which Connect waits for.
	ConnectTimeout time.Duration
	// SkipJoin makes Connect return once the definitions and (with
	// RequestMedia) all media have arrived, without sending CLIENT_READY. The
	// server then never creates the player: no "joined the game" / "left the
	// game" messages, no on_joinplayer, no map blocks or chat. Login and
	// registration still happen. Use it to fetch node definitions and media
	// without showing up in the game.
	SkipJoin bool
}

// Event is one packet that was sent or received.
type Event struct {
	Time     time.Time
	Dir      protocol.Direction
	Channel  uint8
	Reliable bool // only meaningful for client->server packets
	Packet   protocol.Packet
	// Raw is the wire form of the packet (u16 command + data), without transport framing.
	Raw []byte
	// Err is set if a received packet could not be decoded; Packet is then *protocol.Unknown.
	Err error
}

// Errors returned by Connect and Wait.
var (
	// ErrNotRegistered means the account does not exist and Config.Register is false.
	ErrNotRegistered = errors.New("account does not exist on this server (enable registration to create it)")
)

// AccessDeniedError is returned when the server refuses or ends the session.
type AccessDeniedError struct {
	Code   uint8
	Reason string
}

func (e *AccessDeniedError) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = accessDeniedText(e.Code)
	}
	return "access denied: " + reason
}

func accessDeniedText(code uint8) string {
	switch code {
	case protocol.AccessDeniedWrongPassword:
		return "invalid password"
	case protocol.AccessDeniedUnexpectedData:
		return "server received unexpected data"
	case protocol.AccessDeniedSingleplayer:
		return "server is in singleplayer mode"
	case protocol.AccessDeniedWrongVersion:
		return "unsupported protocol version"
	case protocol.AccessDeniedWrongCharsInName:
		return "player name contains disallowed characters"
	case protocol.AccessDeniedWrongName:
		return "player name not allowed"
	case protocol.AccessDeniedTooManyUsers:
		return "too many users"
	case protocol.AccessDeniedEmptyPassword:
		return "empty passwords are not allowed"
	case protocol.AccessDeniedAlreadyConnected:
		return "another client is already connected with this name"
	case protocol.AccessDeniedServerFail:
		return "internal server error"
	case protocol.AccessDeniedShutdown:
		return "server shutting down"
	case protocol.AccessDeniedCrash:
		return "server crashed"
	}
	return fmt.Sprintf("denied (code %d)", code)
}

type state int

const (
	stateInit     state = iota // INIT sent, waiting for HELLO
	stateAuth                  // authenticating
	stateAccepted              // authenticated, loading definitions
	stateLoaded                // definitions and media received, staying out of the game (SkipJoin)
	stateJoined                // CLIENT_READY sent
)

// Client is a headless Luanti client. Create it with New.
type Client struct {
	cfg  Config
	conn *transport.Conn

	sendMu sync.Mutex

	mu       sync.Mutex // guards the fields below
	st       state
	protoVer uint16
	serVer   uint8 // map block serialization version from HELLO
	srpUser  *srp.User
	mech     uint32
	mapSeed  uint64
	pos      protocol.V3F // BS units
	yaw      float32
	pitch    float32
	keys     uint32
	hp       uint16
	privs    []string
	players  map[string]bool
	tod      uint16

	nodeDefs     []protocol.NodeDefinition
	announced    map[string][20]byte // media hashes announced by the server
	awaitBunches int                 // media bunches still expected (-1: not waiting)
	gotBunches   map[uint16]bool

	handlers map[reflect.Type][]func(protocol.Packet)
	anyH     []func(Event)

	blocks *mapdump.Dumper // nil without MapblockDir

	joined   chan struct{}
	joinOnce sync.Once
	done     chan struct{}
	err      error
	closing  chan struct{}
	started  bool
}

// New creates a client. Nothing happens until Connect is called.
func New(cfg Config) *Client {
	if cfg.MediaDir != "" {
		cfg.RequestMedia = true
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	if cfg.ClientVersion == "" {
		cfg.ClientVersion = "5.18.0-go"
	}
	if cfg.PositionInterval <= 0 {
		cfg.PositionInterval = time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 20 * time.Second
	}
	return &Client{
		cfg:          cfg,
		protoVer:     protocol.LatestProtocolVersion,
		players:      map[string]bool{},
		handlers:     map[reflect.Type][]func(protocol.Packet){},
		awaitBunches: -1,
		gotBunches:   map[uint16]bool{},
		announced:    map[string][20]byte{},
		joined:       make(chan struct{}),
		done:         make(chan struct{}),
		closing:      make(chan struct{}),
	}
}

// Handle registers fn to be called for every received packet of type T, e.g.
//
//	client.Handle(c, func(p *protocol.ToClientHP) { ... })
//
// Handlers run on the client's receive loop in registration order; they may
// call Send and friends but must not block for long. Register handlers before
// Connect.
func Handle[T protocol.Packet](c *Client, fn func(T)) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[t] = append(c.handlers[t], func(p protocol.Packet) { fn(p.(T)) })
}

// OnPacket registers fn for every packet in both directions, regardless of Filter.
func (c *Client) OnPacket(fn func(Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.anyH = append(c.anyH, fn)
}

// OnChat registers fn for incoming chat messages.
func (c *Client) OnChat(fn func(*protocol.ToClientChatMessage)) { Handle(c, fn) }

// Connect dials the server, authenticates (registering if configured) and
// joins the game. It returns once the server has accepted the client and sent
// the initial player state (position, privileges, HP), after which packets
// keep flowing until Close is called or the server disconnects (see Wait).
// With Config.SkipJoin it returns before joining, as soon as the definitions
// and media have arrived.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("client: Connect called twice")
	}
	c.started = true
	c.mu.Unlock()

	if c.cfg.Address == "" || c.cfg.Username == "" {
		return errors.New("client: Address and Username are required")
	}
	conn, err := transport.Dial(c.cfg.Address, transport.Options{OnDatagram: c.cfg.OnDatagram}, c.cfg.ConnectTimeout)
	if err != nil {
		return err
	}
	if c.cfg.MapblockDir != "" {
		// created before the receive loop starts, closed by finish
		if c.blocks, err = mapdump.New(c.cfg.MapblockDir, c.cfg.Logf); err != nil {
			_ = conn.Close()
			return err
		}
	}
	c.conn = conn
	go c.loop()

	if err := c.sendInit(); err != nil {
		c.Close()
		return err
	}

	timeout := time.NewTimer(c.cfg.ConnectTimeout)
	defer timeout.Stop()
	select {
	case <-c.joined:
		return nil
	case <-c.done:
		return c.err
	case <-timeout.C:
		c.Close()
		return errors.New("client: timed out joining the server")
	case <-ctx.Done():
		c.Close()
		return ctx.Err()
	}
}

// Wait blocks until the connection has ended and returns why: nil after Close,
// an *AccessDeniedError if the server kicked us, or a transport error.
func (c *Client) Wait() error {
	<-c.done
	return c.err
}

// Done is closed when the connection has ended.
func (c *Client) Done() <-chan struct{} { return c.done }

// Close disconnects from the server. It is safe to call from a handler.
func (c *Client) Close() {
	select {
	case <-c.closing:
	default:
		close(c.closing)
	}
	if c.conn != nil {
		select {
		case <-c.done:
		case <-time.After(2 * time.Second): // e.g. called from inside a handler
		}
	}
}

// Send encodes and sends a client->server packet using the channel and
// reliability Luanti clients use for it.
func (c *Client) Send(p protocol.Packet) error {
	if p.Direction() != protocol.ToServer {
		return fmt.Errorf("client: %s is not a client->server packet", protocol.Name(p))
	}
	if c.conn == nil {
		return errors.New("client: not connected")
	}
	c.mu.Lock()
	ver := c.protoVer
	c.mu.Unlock()
	raw, err := protocol.Encode(p, ver)
	if err != nil {
		return err
	}
	tr := protocol.ToServerTransport(p.Command())
	c.sendMu.Lock()
	err = c.conn.Send(tr.Channel, tr.Reliable, raw)
	c.sendMu.Unlock()
	// Reported after sending and outside the lock so that taps and handlers
	// may themselves call Send.
	c.emit(Event{Time: time.Now(), Dir: protocol.ToServer, Channel: tr.Channel, Reliable: tr.Reliable, Packet: p, Raw: raw})
	return err
}

// Chat sends a chat message or a server command such as "/status".
func (c *Client) Chat(message string) error {
	return c.Send(&protocol.ToServerChatMessage{Message: message})
}

// MoveTo sets the player's position (in nodes) and reports it to the server.
func (c *Client) MoveTo(x, y, z float32) error {
	c.mu.Lock()
	c.pos = protocol.V3F{X: x * 10, Y: y * 10, Z: z * 10}
	c.mu.Unlock()
	return c.SendPosition()
}

// Look sets the viewing direction in degrees and reports it to the server.
func (c *Client) Look(yaw, pitch float32) error {
	c.mu.Lock()
	c.yaw, c.pitch = yaw, pitch
	c.mu.Unlock()
	return c.SendPosition()
}

// SetKeys sets the pressed controls (protocol.Key* bits) and reports them.
func (c *Client) SetKeys(keys uint32) error {
	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()
	return c.SendPosition()
}

// SendPosition reports the current position, view direction and controls.
func (c *Client) SendPosition() error {
	c.mu.Lock()
	p := &protocol.ToServerPlayerPos{}
	p.Position = protocol.V3S32{X: int32(c.pos.X * 100), Y: int32(c.pos.Y * 100), Z: int32(c.pos.Z * 100)}
	p.Pitch = int32(c.pitch * 100)
	p.Yaw = int32(c.yaw * 100)
	p.Keys = c.keys
	p.Fov = 80
	p.WantedRange = c.cfg.ViewRange
	c.mu.Unlock()
	return c.Send(p)
}

// SelectItem selects a hotbar slot (0-based).
func (c *Client) SelectItem(slot uint16) error {
	return c.Send(&protocol.ToServerPlayerItem{Item: slot})
}

// Respawn asks the server to respawn the player after death.
func (c *Client) Respawn() error {
	return c.Send(&protocol.ToServerInventoryFields{
		Formname: "__builtin:death",
		Fields:   []protocol.FormField{{Name: "btn_respawn", Value: ""}},
	})
}

// Position returns the player's position in nodes.
func (c *Client) Position() (x, y, z float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pos.X / 10, c.pos.Y / 10, c.pos.Z / 10
}

// HP returns the player's last reported hit points.
func (c *Client) HP() uint16 { c.mu.Lock(); defer c.mu.Unlock(); return c.hp }

// Privileges returns the player's privileges.
func (c *Client) Privileges() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.privs...)
}

// Players returns the names of the players currently online.
func (c *Client) Players() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.players))
	for n := range c.players {
		out = append(out, n)
	}
	return out
}

// TimeOfDay returns the last reported time of day (0..23999).
func (c *Client) TimeOfDay() uint16 { c.mu.Lock(); defer c.mu.Unlock(); return c.tod }

// MapSeed returns the map seed announced by the server.
func (c *Client) MapSeed() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.mapSeed }

// ProtocolVersion returns the protocol version the server speaks.
func (c *Client) ProtocolVersion() uint16 { c.mu.Lock(); defer c.mu.Unlock(); return c.protoVer }

// --- internals ---

func (c *Client) emit(ev Event) {
	c.mu.Lock()
	hs := c.anyH
	c.mu.Unlock()
	for _, h := range hs {
		h(ev)
	}
	if c.cfg.Tap != nil && c.cfg.Filter.Match(ev.Dir, ev.Packet) {
		c.cfg.Tap(ev)
	}
}

func (c *Client) sendInit() error {
	return c.Send(&protocol.ToServerInit{
		SerializationVersion: protocol.SerializationVersion,
		MinProtocol:          37,
		MaxProtocol:          protocol.LatestProtocolVersion,
		PlayerName:           c.cfg.Username,
	})
}

func (c *Client) finish(err error) {
	c.mu.Lock()
	select {
	case <-c.done:
		c.mu.Unlock()
		return
	default:
	}
	c.err = err
	c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.blocks != nil {
		c.blocks.Close() // write the queued blocks before Wait returns
	}
	close(c.done)
}

func (c *Client) loop() {
	initTick := time.NewTicker(500 * time.Millisecond)
	posTick := time.NewTicker(c.cfg.PositionInterval)
	readyTimer := time.NewTimer(time.Hour)
	readyTimer.Stop()
	defer initTick.Stop()
	defer posTick.Stop()
	defer readyTimer.Stop()

	for {
		select {
		case <-c.closing:
			c.finish(nil)
			return
		case <-c.conn.Done():
			err := c.conn.Err()
			c.finish(err)
			return
		case m, ok := <-c.conn.Messages():
			if !ok {
				c.finish(c.conn.Err())
				return
			}
			if err := c.handle(m, readyTimer); err != nil {
				c.finish(err)
				return
			}
		case <-initTick.C:
			c.mu.Lock()
			init := c.st == stateInit
			c.mu.Unlock()
			if init {
				_ = c.sendInit() // INIT is unreliable; retry until HELLO arrives
			}
		case <-posTick.C:
			c.mu.Lock()
			joined := c.st == stateJoined
			c.mu.Unlock()
			if joined {
				_ = c.SendPosition()
			}
		case <-readyTimer.C:
			// the server never announced media; join anyway
			if err := c.sendReady(); err != nil {
				c.finish(err)
				return
			}
		}
	}
}

func (c *Client) handle(m transport.Message, readyTimer *time.Timer) error {
	c.mu.Lock()
	ver := c.protoVer
	c.mu.Unlock()
	pkt, derr := protocol.DecodeToClient(m.Payload, ver)
	c.emit(Event{Time: time.Now(), Dir: protocol.ToClient, Channel: m.Channel, Packet: pkt, Raw: m.Payload, Err: derr})
	if derr != nil {
		return nil // keep going; the event above reports the problem
	}

	switch p := pkt.(type) {
	case *protocol.ToClientHello:
		if err := c.onHello(p); err != nil {
			return err
		}
	case *protocol.ToClientSRPBytesSB:
		c.mu.Lock()
		u := c.srpUser
		c.mu.Unlock()
		if u == nil {
			return errors.New("client: unexpected SRP challenge")
		}
		mproof, err := u.ProcessChallenge(p.Salt, p.B)
		if err != nil {
			return err
		}
		if err := c.Send(&protocol.ToServerSRPBytesM{M: mproof}); err != nil {
			return err
		}
	case *protocol.ToClientAuthAccept:
		c.mu.Lock()
		c.mapSeed = p.MapSeed
		c.st = stateAccepted
		c.srpUser = nil
		c.mu.Unlock()
		if err := c.Send(&protocol.ToServerInit2{Lang: c.cfg.Language}); err != nil {
			return err
		}
		// fallback for servers that never announce media
		readyTimer.Reset(10 * time.Second)
	case *protocol.ToClientAccessDenied:
		return &AccessDeniedError{Code: p.Code, Reason: p.Reason}
	case *protocol.ToClientAccessDeniedLegacy:
		return &AccessDeniedError{Code: protocol.AccessDeniedCustomString, Reason: p.Reason}
	case *protocol.ToClientAnnounceMedia:
		// the announcement decides when to send CLIENT_READY: right away, or
		// after the last requested media bunch, however long the download takes
		readyTimer.Stop()
		var names []string
		c.mu.Lock()
		for _, f := range p.Files {
			c.announced[f.Name] = f.SHA1
		}
		c.mu.Unlock()
		if c.cfg.RequestMedia {
			for _, f := range p.Files {
				if !c.haveMedia(f.Name, f.SHA1) {
					names = append(names, f.Name)
				}
			}
		}
		if len(names) > 0 {
			c.mu.Lock()
			c.awaitBunches = 0
			c.mu.Unlock()
			if err := c.Send(&protocol.ToServerRequestMedia{Files: names}); err != nil {
				return err
			}
		} else if err := c.sendReady(); err != nil {
			return err
		}
	case *protocol.ToClientNodeDef:
		c.onNodeDefs(p)
	case *protocol.ToClientMedia:
		c.saveMedia(p)
		c.mu.Lock()
		wait := c.awaitBunches >= 0
		if wait {
			c.gotBunches[p.BunchIndex] = true
			c.awaitBunches = int(p.NumBunches) - len(c.gotBunches)
		}
		allDone := wait && c.awaitBunches <= 0
		c.mu.Unlock()
		if allDone {
			if err := c.sendReady(); err != nil {
				return err
			}
		}
	case *protocol.ToClientMovePlayer:
		c.mu.Lock()
		c.pos, c.pitch, c.yaw = p.Pos, p.Pitch, p.Yaw
		c.mu.Unlock()
	case *protocol.ToClientMovePlayerRel:
		c.mu.Lock()
		c.pos.X += p.AddedPos.X
		c.pos.Y += p.AddedPos.Y
		c.pos.Z += p.AddedPos.Z
		c.mu.Unlock()
	case *protocol.ToClientHP:
		c.mu.Lock()
		c.hp = p.HP
		joined := c.st == stateJoined
		c.mu.Unlock()
		if joined {
			c.signalJoined()
		}
	case *protocol.ToClientPrivileges:
		c.mu.Lock()
		c.privs = p.Privileges
		c.mu.Unlock()
	case *protocol.ToClientTimeOfDay:
		c.mu.Lock()
		c.tod = p.Time
		c.mu.Unlock()
	case *protocol.ToClientUpdatePlayerList:
		c.mu.Lock()
		switch p.Type {
		case protocol.PlayerListInit:
			c.players = map[string]bool{}
			fallthrough
		case protocol.PlayerListAdd:
			for _, n := range p.Players {
				c.players[n] = true
			}
		case protocol.PlayerListRemove:
			for _, n := range p.Players {
				delete(c.players, n)
			}
		}
		c.mu.Unlock()
	case *protocol.ToClientBlockData:
		if c.blocks != nil {
			c.mu.Lock()
			serVer := c.serVer
			c.mu.Unlock()
			c.blocks.Block(serVer, p.Pos, p.Data)
		}
		if !c.cfg.DisableBlockAck {
			if err := c.Send(&protocol.ToServerGotBlocks{Blocks: []protocol.V3S16{p.Pos}}); err != nil {
				return err
			}
		}
	}

	c.mu.Lock()
	hs := c.handlers[reflect.TypeOf(pkt)]
	c.mu.Unlock()
	for _, h := range hs {
		h(pkt)
	}
	return nil
}

// NodeDefs returns the node definitions received from the server (nil until
// they arrive, which happens right after login).
func (c *Client) NodeDefs() []protocol.NodeDefinition {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodeDefs
}

func (c *Client) onNodeDefs(p *protocol.ToClientNodeDef) {
	c.mu.Lock()
	ver := c.protoVer
	c.mu.Unlock()
	defs, err := p.NodeDefs(ver)
	if err != nil {
		c.cfg.Logf("nodedefs: %v", err)
		return
	}
	c.mu.Lock()
	c.nodeDefs = defs
	c.mu.Unlock()
	if c.blocks != nil {
		c.blocks.SetNodeDefs(defs)
	}
	if c.cfg.NodeDefsFile != "" {
		write := WriteNodeDefsJSON
		if c.cfg.NodeDefsFormat == "viewer" {
			write = func(path string, _ uint16, defs []protocol.NodeDefinition) error {
				return WriteViewerNodeDefsJSON(path, defs)
			}
		}
		if err := write(c.cfg.NodeDefsFile, ver, defs); err != nil {
			c.cfg.Logf("nodedefs: %v", err)
		}
	}
}

// WriteNodeDefsJSON writes node definitions to path as indented JSON of the
// form {"protocol_version": N, "count": N, "nodes": [...]}. The file is
// replaced atomically.
func WriteNodeDefsJSON(path string, protoVer uint16, defs []protocol.NodeDefinition) error {
	doc := struct {
		ProtocolVersion uint16                    `json:"protocol_version"`
		Count           int                       `json:"count"`
		Nodes           []protocol.NodeDefinition `json:"nodes"`
	}{protoVer, len(defs), defs}
	return writeJSONFile(path, doc)
}

// WriteViewerNodeDefsJSON writes node definitions to path in the schema of
// luanti-viewer's nodedefs.json: an object of node name -> definition (see
// package viewerdef). The file is replaced atomically.
func WriteViewerNodeDefsJSON(path string, defs []protocol.NodeDefinition) error {
	return writeJSONFile(path, viewerdef.Convert(defs))
}

// writeJSONFile writes v as indented JSON, replacing path atomically.
func writeJSONFile(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// validMediaName reports whether name is safe to use as a file name: Luanti
// only allows [A-Za-z0-9_.-], which rules out path separators.
func validMediaName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
		default:
			return false
		}
	}
	return true
}

// haveMedia reports whether MediaDir already holds the file with this hash.
func (c *Client) haveMedia(name string, sum [20]byte) bool {
	if c.cfg.MediaDir == "" || !validMediaName(name) {
		return false
	}
	b, err := os.ReadFile(filepath.Join(c.cfg.MediaDir, name))
	return err == nil && sha1.Sum(b) == sum
}

// saveMedia writes received media files into MediaDir.
func (c *Client) saveMedia(p *protocol.ToClientMedia) {
	if c.cfg.MediaDir == "" {
		return
	}
	if err := os.MkdirAll(c.cfg.MediaDir, 0o755); err != nil {
		c.cfg.Logf("media: %v", err)
		return
	}
	for _, f := range p.Files {
		if !validMediaName(f.Name) {
			c.cfg.Logf("media: refusing unsafe file name %q", f.Name)
			continue
		}
		c.mu.Lock()
		want, known := c.announced[f.Name]
		c.mu.Unlock()
		if !known {
			c.cfg.Logf("media: %q was not announced, skipping", f.Name)
			continue
		}
		if sha1.Sum(f.Data) != want {
			c.cfg.Logf("media: hash mismatch for %q, skipping", f.Name)
			continue
		}
		if err := os.WriteFile(filepath.Join(c.cfg.MediaDir, f.Name), f.Data, 0o644); err != nil {
			c.cfg.Logf("media: %v", err)
		}
	}
}

func (c *Client) onHello(h *protocol.ToClientHello) error {
	c.mu.Lock()
	c.protoVer = h.ProtocolVersion
	c.serVer = h.SerializationVersion
	c.st = stateAuth
	c.mu.Unlock()

	name, pw := c.cfg.Username, c.cfg.Password
	switch {
	case h.AuthMethods&protocol.AuthFirstSRP != 0:
		// The server does not know this name yet.
		if !c.cfg.Register {
			return ErrNotRegistered
		}
		salt, verifier, err := srp.NewVerifier(name, pw)
		if err != nil {
			return err
		}
		c.setMech(protocol.AuthFirstSRP)
		return c.Send(&protocol.ToServerFirstSRP{Salt: salt, Verifier: verifier, EmptyPassword: pw == ""})
	case h.AuthMethods&protocol.AuthSRP != 0, h.AuthMethods&protocol.AuthLegacyPassword != 0:
		basedOn := uint8(1)
		mech := protocol.AuthSRP
		if h.AuthMethods&protocol.AuthSRP == 0 {
			pw = srp.LegacyPassword(name, pw)
			basedOn = 0
			mech = protocol.AuthLegacyPassword
		}
		u, err := srp.NewUser(name, pw)
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.srpUser = u
		c.mech = mech
		c.mu.Unlock()
		return c.Send(&protocol.ToServerSRPBytesA{A: u.BytesA(), BasedOn: basedOn})
	}
	return fmt.Errorf("client: server offers no supported authentication method (%#x)", h.AuthMethods)
}

func (c *Client) setMech(m uint32) { c.mu.Lock(); c.mech = m; c.mu.Unlock() }

func (c *Client) sendReady() error {
	c.mu.Lock()
	if c.st != stateAccepted {
		c.mu.Unlock()
		return nil
	}
	if c.cfg.SkipJoin {
		// everything up to CLIENT_READY has arrived; stay out of the game
		c.st = stateLoaded
		c.mu.Unlock()
		c.joinOnce.Do(func() { close(c.joined) })
		return nil
	}
	c.st = stateJoined
	c.mu.Unlock()
	err := c.Send(&protocol.ToServerClientReady{
		Major: 5, Minor: 18, Patch: 0,
		Version:         c.cfg.ClientVersion,
		FormspecVersion: 12,
	})
	if err != nil {
		return err
	}
	// The server answers CLIENT_READY with position, privileges, inventory and
	// finally HP; Connect returns once that has arrived (or after a short wait,
	// for servers that never send HP).
	time.AfterFunc(3*time.Second, c.signalJoined)
	return c.SendPosition()
}

// signalJoined marks the client as in-game. The server drops
// TOSERVER_UPDATE_CLIENT_INFO until it has processed CLIENT_READY (the client
// must be "active"), so it is sent here rather than right after INIT2.
func (c *Client) signalJoined() {
	c.joinOnce.Do(func() {
		_ = c.Send(&protocol.ToServerUpdateClientInfo{
			RenderTargetWidth: 1280, RenderTargetHeight: 720,
			GUIScaling: 1, HUDScaling: 1, MaxFormspecWidth: 20, MaxFormspecHeight: 12,
		})
		close(c.joined)
	})
}
