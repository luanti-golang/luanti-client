// Package transport implements Luanti's UDP transport protocol ("MTP"):
// peer-id handshake, three channels, reliable delivery with acknowledgements
// and retransmission, and split-packet reassembly.
//
// It deals in opaque payloads. A payload is what Luanti calls a network
// packet: a big-endian u16 command followed by the command's data. Decoding
// payloads into typed structs is the job of package protocol.
package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	protocolID  = 0x4f457403
	baseHeader  = 7
	maxDatagram = 512 // what Luanti clients and servers use by default

	seqInitial = 65500
	seqMax     = 65535

	// ChannelCount is the number of channels of a connection.
	ChannelCount = 3

	// PeerIDServer is the fixed peer id of the server.
	PeerIDServer = 1

	maxIncomingWindow = 0x8000
	sendWindow        = 256

	pingInterval     = 5 * time.Second
	inactivityLimit  = 30 * time.Second
	splitExpiry      = 30 * time.Second
	minResendTimeout = 500 * time.Millisecond
)

// Packet types in the MTP header.
const (
	TypeControl  = 0
	TypeOriginal = 1
	TypeSplit    = 2
	TypeReliable = 3
)

// Control types.
const (
	ControlAck      = 0
	ControlSetPeer  = 1
	ControlPing     = 2
	ControlDisco    = 3
	controlTypeSize = 2
)

// Direction says in which direction a datagram or packet travelled.
type Direction int

const (
	// ToServer marks traffic sent by us.
	ToServer Direction = iota
	// ToClient marks traffic received from the server.
	ToClient
)

func (d Direction) String() string {
	if d == ToServer {
		return "->server"
	}
	return "<-server"
}

// Message is a fully reassembled payload received from the server.
type Message struct {
	Channel uint8
	// Payload is a u16 command followed by command data.
	Payload []byte
}

// Options configure a connection.
type Options struct {
	// OnDatagram, if set, is called for every raw UDP datagram that is sent or
	// received. The slice must not be retained or modified.
	OnDatagram func(dir Direction, datagram []byte)
}

// ErrClosed is returned for operations on a closed connection.
var ErrClosed = errors.New("transport: connection closed")

type outPacket struct {
	data     []byte // complete datagram
	seq      uint16
	sentAt   time.Time // last (re)send
	firstAt  time.Time
	resends  int
	acked    bool
	hasSent  bool
	original bool
}

type splitBuf struct {
	count  int
	chunks map[uint16][]byte
	added  time.Time
}

type channel struct {
	nextOut      uint16
	nextSplitOut uint16
	nextIn       uint16
	inflight     []*outPacket
	pending      [][]byte // inner packets waiting for window space
	incoming     map[uint16][]byte
	splits       map[uint16]*splitBuf
}

// Conn is a client-side connection to a Luanti server.
type Conn struct {
	udp  *net.UDPConn
	opts Options

	mu       sync.Mutex
	peerID   uint16
	ch       [ChannelCount]*channel
	srtt     time.Duration
	lastRecv time.Time
	lastSend time.Time
	closed   bool
	err      error

	peerCh chan struct{} // closed once the server assigned our peer id
	peerOK bool

	msgs chan Message
	done chan struct{}
}

// Dial opens a connection to the given host:port and performs the MTP
// handshake. It returns once the server has assigned a peer id.
func Dial(address string, opts Options, timeout time.Duration) (*Conn, error) {
	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	udp, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		udp:      udp,
		opts:     opts,
		srtt:     minResendTimeout,
		lastRecv: time.Now(),
		peerCh:   make(chan struct{}),
		msgs:     make(chan Message, 1024),
		done:     make(chan struct{}),
	}
	for i := range c.ch {
		c.ch[i] = &channel{
			nextOut: seqInitial, nextSplitOut: seqInitial, nextIn: seqInitial,
			incoming: map[uint16][]byte{}, splits: map[uint16]*splitBuf{},
		}
	}

	go c.readLoop()
	go c.timerLoop()

	// The handshake packet is an empty reliable packet from peer id 0.
	c.mu.Lock()
	c.ch[0].pending = append(c.ch[0].pending, []byte{TypeOriginal})
	c.flushLocked(0)
	c.mu.Unlock()

	select {
	case <-c.peerCh:
		return c, nil
	case <-c.done:
		return nil, c.Err()
	case <-time.After(timeout):
		c.fail(errors.New("transport: timed out waiting for the server"))
		return nil, c.Err()
	}
}

// Messages returns the channel on which received payloads are delivered. It
// is closed when the connection ends; check Err for the reason.
func (c *Conn) Messages() <-chan Message { return c.msgs }

// Done is closed when the connection has ended.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Err returns why the connection ended (nil if closed locally or still open).
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// PeerID returns the peer id assigned by the server.
func (c *Conn) PeerID() uint16 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peerID
}

// Send transmits a payload (u16 command + data) on the given channel.
func (c *Conn) Send(channelNum uint8, reliable bool, payload []byte) error {
	if channelNum >= ChannelCount {
		return fmt.Errorf("transport: bad channel %d", channelNum)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if reliable {
		c.queueReliableLocked(channelNum, payload)
		return nil
	}
	chunk := maxDatagram - baseHeader
	for _, inner := range c.autoSplitLocked(channelNum, payload, chunk) {
		c.rawSendLocked(c.buildLocked(channelNum, inner))
	}
	return nil
}

// Close sends a disconnect notice and shuts the connection down.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.rawSendLocked(c.buildLocked(0, []byte{TypeControl, ControlDisco}))
	c.mu.Unlock()
	c.fail(nil)
	return nil
}

// --- sending internals ---

func (c *Conn) buildLocked(channelNum uint8, inner []byte) []byte {
	b := make([]byte, baseHeader+len(inner))
	binary.BigEndian.PutUint32(b[0:], protocolID)
	binary.BigEndian.PutUint16(b[4:], c.peerID)
	b[6] = channelNum
	copy(b[baseHeader:], inner)
	return b
}

func (c *Conn) rawSendLocked(datagram []byte) {
	if c.opts.OnDatagram != nil {
		c.opts.OnDatagram(ToServer, datagram)
	}
	c.lastSend = time.Now()
	_, _ = c.udp.Write(datagram)
}

// autoSplitLocked wraps data into ORIGINAL or (for big data) SPLIT packets.
// chunkMax is the maximum size of an inner packet.
func (c *Conn) autoSplitLocked(channelNum uint8, data []byte, chunkMax int) [][]byte {
	if len(data)+1 <= chunkMax {
		return [][]byte{append([]byte{TypeOriginal}, data...)}
	}
	const splitHeader = 7
	per := chunkMax - splitHeader
	count := (len(data) + per - 1) / per
	cs := c.ch[channelNum]
	seq := cs.nextSplitOut
	cs.nextSplitOut++
	out := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		lo, hi := i*per, (i+1)*per
		if hi > len(data) {
			hi = len(data)
		}
		p := make([]byte, splitHeader+hi-lo)
		p[0] = TypeSplit
		binary.BigEndian.PutUint16(p[1:], seq)
		binary.BigEndian.PutUint16(p[3:], uint16(count))
		binary.BigEndian.PutUint16(p[5:], uint16(i))
		copy(p[splitHeader:], data[lo:hi])
		out = append(out, p)
	}
	return out
}

func (c *Conn) queueReliableLocked(channelNum uint8, payload []byte) {
	inners := c.autoSplitLocked(channelNum, payload, maxDatagram-baseHeader-3)
	cs := c.ch[channelNum]
	cs.pending = append(cs.pending, inners...)
	c.flushLocked(channelNum)
}

func (c *Conn) flushLocked(channelNum uint8) {
	cs := c.ch[channelNum]
	for len(cs.pending) > 0 && len(cs.inflight) < sendWindow {
		inner := cs.pending[0]
		cs.pending = cs.pending[1:]
		seq := cs.nextOut
		cs.nextOut++
		rel := make([]byte, 3+len(inner))
		rel[0] = TypeReliable
		binary.BigEndian.PutUint16(rel[1:], seq)
		copy(rel[3:], inner)
		now := time.Now()
		p := &outPacket{data: c.buildLocked(channelNum, rel), seq: seq, sentAt: now, firstAt: now}
		cs.inflight = append(cs.inflight, p)
		c.rawSendLocked(p.data)
	}
}

func (c *Conn) sendAckLocked(channelNum uint8, seq uint16) {
	c.rawSendLocked(c.buildLocked(channelNum, []byte{TypeControl, ControlAck, byte(seq >> 8), byte(seq)}))
}

// --- receiving internals ---

func (c *Conn) readLoop() {
	defer close(c.msgs) // the read loop is the only sender on msgs
	buf := make([]byte, 65536)
	for {
		n, err := c.udp.Read(buf)
		if err != nil {
			select {
			case <-c.done:
			default:
				c.fail(err)
			}
			return
		}
		c.handleDatagram(buf[:n])
	}
}

func (c *Conn) handleDatagram(d []byte) {
	if c.opts.OnDatagram != nil {
		c.opts.OnDatagram(ToClient, d)
	}
	if len(d) < baseHeader+1 || binary.BigEndian.Uint32(d) != protocolID {
		return
	}
	channelNum := d[6]
	if channelNum >= ChannelCount {
		return
	}
	var deliver []Message
	c.mu.Lock()
	c.lastRecv = time.Now()
	c.processLocked(channelNum, d[baseHeader:], false, &deliver)
	c.mu.Unlock()
	for _, m := range deliver {
		select {
		case c.msgs <- m:
		case <-c.done:
			return
		}
	}
}

func (c *Conn) processLocked(channelNum uint8, p []byte, reliable bool, out *[]Message) {
	if len(p) == 0 {
		return
	}
	cs := c.ch[channelNum]
	switch p[0] {
	case TypeControl:
		if len(p) < controlTypeSize {
			return
		}
		switch p[1] {
		case ControlAck:
			if len(p) < 4 {
				return
			}
			c.handleAckLocked(channelNum, binary.BigEndian.Uint16(p[2:]))
		case ControlSetPeer:
			if len(p) < 4 {
				return
			}
			if c.peerID == 0 {
				c.peerID = binary.BigEndian.Uint16(p[2:])
				c.peerOK = true
				close(c.peerCh)
			}
		case ControlPing:
		case ControlDisco:
			go c.fail(errors.New("transport: server closed the connection"))
		}
	case TypeOriginal:
		if len(p) > 1 {
			*out = append(*out, Message{Channel: channelNum, Payload: append([]byte(nil), p[1:]...)})
		}
	case TypeSplit:
		if len(p) < 7 {
			return
		}
		seq := binary.BigEndian.Uint16(p[1:])
		count := int(binary.BigEndian.Uint16(p[3:]))
		num := binary.BigEndian.Uint16(p[5:])
		if count == 0 || int(num) >= count {
			return
		}
		sb := cs.splits[seq]
		if sb == nil {
			sb = &splitBuf{count: count, chunks: map[uint16][]byte{}, added: time.Now()}
			cs.splits[seq] = sb
		}
		if sb.count != count {
			return
		}
		sb.chunks[num] = append([]byte(nil), p[7:]...)
		if len(sb.chunks) == sb.count {
			var all []byte
			for i := 0; i < sb.count; i++ {
				all = append(all, sb.chunks[uint16(i)]...)
			}
			delete(cs.splits, seq)
			if len(all) > 0 {
				*out = append(*out, Message{Channel: channelNum, Payload: all})
			}
		}
	case TypeReliable:
		if reliable || len(p) < 4 {
			return
		}
		seq := binary.BigEndian.Uint16(p[1:])
		if seqInWindow(seq, cs.nextIn, maxIncomingWindow) {
			c.sendAckLocked(channelNum, seq)
		} else if seqHigher(cs.nextIn, seq) {
			c.sendAckLocked(channelNum, seq) // our ack got lost
			return
		} else {
			return // too far in the future
		}
		if seq != cs.nextIn {
			if _, dup := cs.incoming[seq]; !dup {
				cs.incoming[seq] = append([]byte(nil), p[3:]...)
			}
			return
		}
		cs.nextIn++
		c.processLocked(channelNum, p[3:], true, out)
		for {
			next, ok := cs.incoming[cs.nextIn]
			if !ok {
				break
			}
			delete(cs.incoming, cs.nextIn)
			cs.nextIn++
			c.processLocked(channelNum, next, true, out)
		}
	}
}

func (c *Conn) handleAckLocked(channelNum uint8, seq uint16) {
	cs := c.ch[channelNum]
	for i, p := range cs.inflight {
		if p.seq != seq {
			continue
		}
		if p.resends == 0 {
			rtt := time.Since(p.firstAt)
			// srtt is the resend timeout: a smoothed 2x RTT, never below the minimum
			c.srtt = (c.srtt*7 + max(rtt*2, minResendTimeout)) / 8
		}
		cs.inflight = append(cs.inflight[:i], cs.inflight[i+1:]...)
		c.flushLocked(channelNum)
		return
	}
}

func (c *Conn) timerLoop() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			return
		case now := <-t.C:
			c.mu.Lock()
			if now.Sub(c.lastRecv) > inactivityLimit {
				c.mu.Unlock()
				c.fail(errors.New("transport: connection timed out"))
				return
			}
			for _, cs := range c.ch {
				for _, p := range cs.inflight {
					timeout := c.srtt * time.Duration(1<<min(p.resends, 4))
					if now.Sub(p.sentAt) >= timeout {
						binary.BigEndian.PutUint16(p.data[4:], c.peerID) // peer id may have changed
						p.sentAt = now
						p.resends++
						c.rawSendLocked(p.data)
					}
				}
				for seq, sb := range cs.splits {
					if now.Sub(sb.added) > splitExpiry {
						delete(cs.splits, seq)
					}
				}
			}
			if c.peerOK && now.Sub(c.lastSend) >= pingInterval {
				c.rawSendLocked(c.buildLocked(0, []byte{TypeControl, ControlPing}))
			}
			c.mu.Unlock()
		}
	}
}

// fail shuts the connection down, recording err as the reason.
func (c *Conn) fail(err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.err = err
	c.mu.Unlock()
	close(c.done)
	_ = c.udp.Close() // unblocks readLoop, which then closes msgs
}

func seqHigher(test, base uint16) bool {
	if test > base {
		return test-base <= seqMax/2
	}
	return base-test > seqMax/2
}

func seqInWindow(seq, next uint16, window int) bool {
	start := int(next)
	end := (start + window) % (seqMax + 1)
	s := int(seq)
	if start < end {
		return s >= start && s < end
	}
	return s < end || s >= start
}
