package protocol

// Highest serialization format version this client speaks.
const SerializationVersion = 29

// ToServerInit is the first packet of a connection.
type ToServerInit struct {
	SerializationVersion uint8
	Compression          uint16 // unused
	MinProtocol          uint16
	MaxProtocol          uint16
	PlayerName           string
}

func (p *ToServerInit) Sync(s *Stream) {
	s.U8(&p.SerializationVersion)
	s.U16(&p.Compression)
	s.U16(&p.MinProtocol)
	s.U16(&p.MaxProtocol)
	s.Str(&p.PlayerName)
}

// ToServerInit2 acknowledges ToClientAuthAccept; afterwards the server sends data.
type ToServerInit2 struct{ Lang string }

func (p *ToServerInit2) Sync(s *Stream) {
	if s.More() {
		s.Str(&p.Lang)
	}
}

// ToServerModChannelJoin joins a mod channel.
type ToServerModChannelJoin struct{ Channel string }

func (p *ToServerModChannelJoin) Sync(s *Stream) { s.Str(&p.Channel) }

// ToServerModChannelLeave leaves a mod channel.
type ToServerModChannelLeave struct{ Channel string }

func (p *ToServerModChannelLeave) Sync(s *Stream) { s.Str(&p.Channel) }

// ToServerModChannelMsg sends a message on a mod channel.
type ToServerModChannelMsg struct{ Channel, Message string }

func (p *ToServerModChannelMsg) Sync(s *Stream) { s.Str(&p.Channel); s.Str(&p.Message) }

// PlayerPosData is the movement state shared by ToServerPlayerPos and
// ToServerInteract. Fixed-point values are multiplied by 100; positions are in
// BS units (10 per node).
type PlayerPosData struct {
	Position          V3S32
	Speed             V3S32
	Pitch             int32
	Yaw               int32
	Keys              uint32
	Fov               uint8 // fov * 80
	WantedRange       uint8 // in map blocks
	CameraInverted    bool
	MovementSpeed     float32
	MovementDirection float32
}

func (d *PlayerPosData) sync(s *Stream) {
	s.V3S32(&d.Position)
	s.V3S32(&d.Speed)
	s.S32(&d.Pitch)
	s.S32(&d.Yaw)
	s.U32(&d.Keys)
	s.U8(&d.Fov)
	s.U8(&d.WantedRange)
	if s.More() {
		s.Bool(&d.CameraInverted)
	}
	if s.More() {
		s.F32(&d.MovementSpeed)
		s.F32(&d.MovementDirection)
	}
}

// Key bits of PlayerPosData.Keys.
const (
	KeyForward uint32 = 1 << iota
	KeyBackward
	KeyLeft
	KeyRight
	KeyJump
	KeyAux1
	KeySneak
	KeyDig
	KeyPlace
	KeyZoom
)

// ToServerPlayerPos reports the player's position and controls.
type ToServerPlayerPos struct{ PlayerPosData }

func (p *ToServerPlayerPos) Sync(s *Stream) { p.PlayerPosData.sync(s) }

// ToServerGotBlocks acknowledges received map blocks.
type ToServerGotBlocks struct{ Blocks []V3S16 }

func (p *ToServerGotBlocks) Sync(s *Stream) {
	Slice8(s, &p.Blocks, func(e *V3S16) { s.V3S16(e) })
}

// ToServerDeletedBlocks reports map blocks the client dropped.
type ToServerDeletedBlocks struct{ Blocks []V3S16 }

func (p *ToServerDeletedBlocks) Sync(s *Stream) {
	Slice8(s, &p.Blocks, func(e *V3S16) { s.V3S16(e) })
}

// ToServerInventoryAction performs an inventory action. Action is Luanti's
// textual format, e.g. "Move current_player main 1 current_player main 2 10".
type ToServerInventoryAction struct{ Action string }

func (p *ToServerInventoryAction) Sync(s *Stream) {
	var b []byte
	if !s.Reading {
		b = []byte(p.Action)
	}
	s.Rest(&b)
	p.Action = string(b)
}

// ToServerChatMessage sends a chat message or a /command.
type ToServerChatMessage struct{ Message string }

func (p *ToServerChatMessage) Sync(s *Stream) { s.WStr(&p.Message) }

// ToServerDamage reports fall damage and similar client-side damage.
type ToServerDamage struct{ Amount uint16 }

func (p *ToServerDamage) Sync(s *Stream) { s.U16(&p.Amount) }

// ToServerPlayerItem selects a hotbar slot.
type ToServerPlayerItem struct{ Item uint16 }

func (p *ToServerPlayerItem) Sync(s *Stream) { s.U16(&p.Item) }

// ToServerRespawnLegacy requests a respawn (servers older than 5.10).
type ToServerRespawnLegacy struct{}

func (p *ToServerRespawnLegacy) Sync(s *Stream) {}

// Interact actions for ToServerInteract.Action.
const (
	InteractStartDigging = iota
	InteractStopDigging
	InteractDiggingCompleted
	InteractPlace
	InteractUse
	InteractActivate
)

// Pointed thing types.
const (
	PointedNothing = iota
	PointedNode
	PointedObject
)

// PointedThing says what the player is pointing at.
type PointedThing struct {
	Type     uint8 // Pointed* constant
	Under    V3S16 // node: the pointed node
	Above    V3S16 // node: the node in front of it
	ObjectID uint16
}

func (t *PointedThing) sync(s *Stream) {
	ver := uint8(0)
	s.U8(&ver)
	s.U8(&t.Type)
	switch t.Type {
	case PointedNode:
		s.V3S16(&t.Under)
		s.V3S16(&t.Above)
	case PointedObject:
		s.U16(&t.ObjectID)
	}
}

// ToServerInteract digs, places or uses an item.
type ToServerInteract struct {
	Action uint8  // Interact* constant
	Item   uint16 // selected hotbar slot
	Target PointedThing
	PlayerPosData
}

func (p *ToServerInteract) Sync(s *Stream) {
	s.U8(&p.Action)
	s.U16(&p.Item)
	// the pointed thing is wrapped in a u32 length-prefixed string
	if s.Reading {
		var raw []byte
		s.LBlob(&raw)
		if s.Err != nil {
			return
		}
		in := NewReader(raw, s.ProtoVer)
		p.Target.sync(in)
		s.Fail(in.Err)
	} else {
		w := NewWriter(s.ProtoVer)
		p.Target.sync(w)
		raw := w.Bytes()
		s.LBlob(&raw)
	}
	p.PlayerPosData.sync(s)
}

// ToServerRemovedSounds tells the server which sounds finished playing.
type ToServerRemovedSounds struct{ IDs []int32 }

func (p *ToServerRemovedSounds) Sync(s *Stream) {
	Slice16(s, &p.IDs, func(e *int32) { s.S32(e) })
}

// FormField is a formspec field.
type FormField struct {
	Name  string
	Value string
}

func syncFields(s *Stream, f *[]FormField) {
	Slice16(s, f, func(e *FormField) { s.Str(&e.Name); s.LStr(&e.Value) })
}

// ToServerNodemetaFields submits a node formspec.
type ToServerNodemetaFields struct {
	Pos      V3S16
	Formname string
	Fields   []FormField
}

func (p *ToServerNodemetaFields) Sync(s *Stream) {
	s.V3S16(&p.Pos)
	s.Str(&p.Formname)
	syncFields(s, &p.Fields)
}

// ToServerInventoryFields submits a player formspec.
type ToServerInventoryFields struct {
	Formname string
	Fields   []FormField
}

func (p *ToServerInventoryFields) Sync(s *Stream) {
	s.Str(&p.Formname)
	syncFields(s, &p.Fields)
}

// ToServerRequestMedia asks the server for media files.
type ToServerRequestMedia struct{ Files []string }

func (p *ToServerRequestMedia) Sync(s *Stream) { s.Strings16(&p.Files) }

// ToServerHaveMedia confirms that dynamic media was loaded.
type ToServerHaveMedia struct{ Tokens []uint32 }

func (p *ToServerHaveMedia) Sync(s *Stream) {
	Slice8(s, &p.Tokens, func(e *uint32) { s.U32(e) })
}

// ToServerClientReady tells the server that the client finished loading.
type ToServerClientReady struct {
	Major, Minor, Patch uint8
	Reserved            uint8
	Version             string // full version string
	FormspecVersion     uint16
}

func (p *ToServerClientReady) Sync(s *Stream) {
	s.U8(&p.Major)
	s.U8(&p.Minor)
	s.U8(&p.Patch)
	s.U8(&p.Reserved)
	s.Str(&p.Version)
	if s.More() {
		s.U16(&p.FormspecVersion)
	}
}

// ToServerFirstSRP registers an account by sending its SRP verifier.
type ToServerFirstSRP struct {
	Salt          []byte
	Verifier      []byte
	EmptyPassword bool
}

func (p *ToServerFirstSRP) Sync(s *Stream) {
	s.Blob(&p.Salt)
	s.Blob(&p.Verifier)
	s.Bool(&p.EmptyPassword)
}

// ToServerSRPBytesA starts an SRP login.
type ToServerSRPBytesA struct {
	A       []byte
	BasedOn uint8 // 0: legacy hash, 1: password
}

func (p *ToServerSRPBytesA) Sync(s *Stream) { s.Blob(&p.A); s.U8(&p.BasedOn) }

// ToServerSRPBytesM answers the server's SRP challenge.
type ToServerSRPBytesM struct{ M []byte }

func (p *ToServerSRPBytesM) Sync(s *Stream) { s.Blob(&p.M) }

// ToServerUpdateClientInfo reports display properties.
type ToServerUpdateClientInfo struct {
	RenderTargetWidth  uint32
	RenderTargetHeight uint32
	GUIScaling         float32
	HUDScaling         float32
	MaxFormspecWidth   float32
	MaxFormspecHeight  float32
	TouchControls      bool
}

func (p *ToServerUpdateClientInfo) Sync(s *Stream) {
	s.U32(&p.RenderTargetWidth)
	s.U32(&p.RenderTargetHeight)
	s.F32(&p.GUIScaling)
	s.F32(&p.HUDScaling)
	s.F32(&p.MaxFormspecWidth)
	s.F32(&p.MaxFormspecHeight)
	if s.More() {
		s.Bool(&p.TouchControls)
	}
}
