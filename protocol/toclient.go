package protocol

import (
	"encoding/base64"
	"errors"
)

// Authentication mechanisms offered in ToClientHello.AuthMethods.
const (
	AuthLegacyPassword uint32 = 1 << 0 // SRP based on the legacy password hash
	AuthSRP            uint32 = 1 << 1 // SRP based on the verifier
	AuthFirstSRP       uint32 = 1 << 2 // first login: register a verifier
)

// Reasons in ToClientAccessDenied.Code.
const (
	AccessDeniedWrongPassword = iota
	AccessDeniedUnexpectedData
	AccessDeniedSingleplayer
	AccessDeniedWrongVersion
	AccessDeniedWrongCharsInName
	AccessDeniedWrongName
	AccessDeniedTooManyUsers
	AccessDeniedEmptyPassword
	AccessDeniedAlreadyConnected
	AccessDeniedServerFail
	AccessDeniedCustomString
	AccessDeniedShutdown
	AccessDeniedCrash
)

// Chat message types in ToClientChatMessage.Type.
const (
	ChatRaw = iota
	ChatNormal
	ChatAnnounce
	ChatSystem
)

// ToClientHello is the server's reply to TOSERVER_INIT.
type ToClientHello struct {
	SerializationVersion uint8
	Compression          uint16 // unused
	ProtocolVersion      uint16
	AuthMethods          uint32 // bit set of Auth* constants
	LegacyName           string // unused
}

func (p *ToClientHello) Sync(s *Stream) {
	s.U8(&p.SerializationVersion)
	s.U16(&p.Compression)
	s.U16(&p.ProtocolVersion)
	s.U32(&p.AuthMethods)
	s.Str(&p.LegacyName)
}

// ToClientAuthAccept tells the client that authentication succeeded.
type ToClientAuthAccept struct {
	PlayerPos       V3F // unused
	MapSeed         uint64
	SendInterval    float32
	SudoAuthMethods uint32
}

func (p *ToClientAuthAccept) Sync(s *Stream) {
	s.V3F(&p.PlayerPos)
	s.U64(&p.MapSeed)
	s.F32(&p.SendInterval)
	s.U32(&p.SudoAuthMethods)
}

// ToClientAcceptSudoMode confirms sudo mode (password change).
type ToClientAcceptSudoMode struct{}

func (p *ToClientAcceptSudoMode) Sync(s *Stream) {}

// ToClientDenySudoMode signals that sudo mode authentication failed.
type ToClientDenySudoMode struct{}

func (p *ToClientDenySudoMode) Sync(s *Stream) {}

// ToClientAccessDenied tells the client that it may not join.
type ToClientAccessDenied struct {
	Code      uint8 // AccessDenied* constant
	Reason    string
	Reconnect bool
}

func (p *ToClientAccessDenied) Sync(s *Stream) {
	s.U8(&p.Code)
	if s.More() {
		s.Str(&p.Reason)
	}
	if s.More() {
		s.Bool(&p.Reconnect)
	}
}

// ToClientBlockData carries one serialized map block.
type ToClientBlockData struct {
	Pos  V3S16
	Data []byte // serialized MapBlock (kept undecoded)
}

func (p *ToClientBlockData) Sync(s *Stream) { s.V3S16(&p.Pos); s.Rest(&p.Data) }

// ToClientAddNode sets a single node.
type ToClientAddNode struct {
	Pos          V3S16
	Node         MapNode
	KeepMetadata bool
}

func (p *ToClientAddNode) Sync(s *Stream) {
	s.V3S16(&p.Pos)
	p.Node.sync(s)
	s.Bool(&p.KeepMetadata)
}

// ToClientRemoveNode removes a single node.
type ToClientRemoveNode struct{ Pos V3S16 }

func (p *ToClientRemoveNode) Sync(s *Stream) { s.V3S16(&p.Pos) }

// ToClientInventory sends the player's inventory in Luanti's text format.
type ToClientInventory struct {
	Inventory     string
	SkipWieldAnim bool
}

func (p *ToClientInventory) Sync(s *Stream) {
	if s.ProtoVer > 51 {
		s.LStr(&p.Inventory)
		s.Bool(&p.SkipWieldAnim)
		return
	}
	var b []byte
	if !s.Reading {
		b = []byte(p.Inventory)
	}
	s.Rest(&b)
	p.Inventory = string(b)
}

// ToClientTimeOfDay sets the time of day.
type ToClientTimeOfDay struct {
	Time  uint16 // 0..23999
	Speed float32
}

func (p *ToClientTimeOfDay) Sync(s *Stream) { s.U16(&p.Time); s.F32(&p.Speed) }

// ToClientCSMRestrictionFlags tells which client-side-mod features are restricted.
type ToClientCSMRestrictionFlags struct {
	Flags     uint64
	NodeRange uint32
}

func (p *ToClientCSMRestrictionFlags) Sync(s *Stream) { s.U64(&p.Flags); s.U32(&p.NodeRange) }

// ToClientPlayerSpeed adds velocity to the player.
type ToClientPlayerSpeed struct{ AddedVel V3F }

func (p *ToClientPlayerSpeed) Sync(s *Stream) { s.V3F(&p.AddedVel) }

// ToClientMediaPush announces a dynamically added media file.
type ToClientMediaPush struct {
	SHA1           []byte // raw 20-byte digest
	Filename       string
	ShouldBeCached bool
	CallbackToken  uint32
}

func (p *ToClientMediaPush) Sync(s *Stream) {
	s.Blob(&p.SHA1)
	s.Str(&p.Filename)
	s.Bool(&p.ShouldBeCached)
	s.U32(&p.CallbackToken)
}

// ToClientChatMessage delivers a chat line.
type ToClientChatMessage struct {
	Version   uint8 // always 1
	Type      uint8 // Chat* constant
	Sender    string
	Message   string
	Timestamp uint64 // unix seconds
}

func (p *ToClientChatMessage) Sync(s *Stream) {
	s.U8(&p.Version)
	s.U8(&p.Type)
	s.WStr(&p.Sender)
	s.WStr(&p.Message)
	s.U64(&p.Timestamp)
}

// ActiveObjectAdd describes an object entering the client's view.
type ActiveObjectAdd struct {
	ID       uint16
	Type     uint8
	InitData []byte
}

// ToClientActiveObjectRemoveAdd adds and removes active objects (entities, players).
type ToClientActiveObjectRemoveAdd struct {
	Removed []uint16
	Added   []ActiveObjectAdd
}

func (p *ToClientActiveObjectRemoveAdd) Sync(s *Stream) {
	Slice16(s, &p.Removed, func(e *uint16) { s.U16(e) })
	Slice16(s, &p.Added, func(e *ActiveObjectAdd) {
		s.U16(&e.ID)
		s.U8(&e.Type)
		s.LBlob(&e.InitData)
	})
}

// ActiveObjectMessage is a command for one active object.
type ActiveObjectMessage struct {
	ID   uint16
	Data []byte
}

// ToClientActiveObjectMessages carries commands for active objects.
type ToClientActiveObjectMessages struct{ Messages []ActiveObjectMessage }

func (p *ToClientActiveObjectMessages) Sync(s *Stream) {
	SliceRest(s, &p.Messages, func(e *ActiveObjectMessage) { s.U16(&e.ID); s.Blob(&e.Data) })
}

// ToClientHP sets the player's hit points.
type ToClientHP struct {
	HP           uint16
	DamageEffect bool
}

func (p *ToClientHP) Sync(s *Stream) {
	s.U16(&p.HP)
	p.DamageEffect = true
	if s.More() {
		s.Bool(&p.DamageEffect)
	}
}

// ToClientMovePlayer teleports the player.
type ToClientMovePlayer struct {
	Pos   V3F // in BS units (10 per node)
	Pitch float32
	Yaw   float32
}

func (p *ToClientMovePlayer) Sync(s *Stream) { s.V3F(&p.Pos); s.F32(&p.Pitch); s.F32(&p.Yaw) }

// ToClientAccessDeniedLegacy is the pre-5.6 variant of ToClientAccessDenied.
type ToClientAccessDeniedLegacy struct{ Reason string }

func (p *ToClientAccessDeniedLegacy) Sync(s *Stream) { s.WStr(&p.Reason) }

// ToClientFov overrides the field of view.
type ToClientFov struct {
	Fov            float32
	IsMultiplier   bool
	TransitionTime float32
}

func (p *ToClientFov) Sync(s *Stream) {
	s.F32(&p.Fov)
	s.Bool(&p.IsMultiplier)
	if s.More() {
		s.F32(&p.TransitionTime)
	}
}

// ToClientDeathscreenLegacy is sent by servers older than 5.10 on death.
type ToClientDeathscreenLegacy struct {
	Unused bool
	Pos    V3F
}

func (p *ToClientDeathscreenLegacy) Sync(s *Stream) {
	if s.More() {
		s.Bool(&p.Unused)
	}
	if s.More() {
		s.V3F(&p.Pos)
	}
}

// MediaFile is one file of a ToClientMedia bunch.
type MediaFile struct {
	Name string
	Data []byte // decompressed
}

// ToClientMedia delivers a bunch of requested media files.
type ToClientMedia struct {
	NumBunches uint16
	BunchIndex uint16
	Files      []MediaFile
}

func (p *ToClientMedia) Sync(s *Stream) {
	s.U16(&p.NumBunches)
	s.U16(&p.BunchIndex)
	Slice32(s, &p.Files, func(f *MediaFile) {
		s.Str(&f.Name)
		if s.ProtoVer < 48 {
			s.LBlob(&f.Data)
			return
		}
		if s.Reading {
			var c []byte
			s.LBlob(&c)
			if s.Err != nil {
				return
			}
			d, err := DecompressZstd(c)
			if err != nil {
				s.Fail(err)
				return
			}
			f.Data = d
		} else {
			c := CompressZstd(f.Data)
			s.LBlob(&c)
		}
	})
}

// ToClientNodeDef carries the node definitions of the game.
type ToClientNodeDef struct {
	Definitions []byte // decompressed serialized NodeDefManager (kept undecoded)
}

func (p *ToClientNodeDef) Sync(s *Stream) { s.compressedBlob(&p.Definitions) }

// MediaAnnounce is one entry of ToClientAnnounceMedia.
type MediaAnnounce struct {
	Name string
	SHA1 [20]byte
}

// ToClientAnnounceMedia lists all media files of the server.
type ToClientAnnounceMedia struct {
	Files      []MediaAnnounce
	RemoteURLs string // comma separated list of remote media servers
}

func (p *ToClientAnnounceMedia) Sync(s *Stream) {
	if s.ProtoVer >= 48 {
		if s.Reading {
			var c []byte
			s.LBlob(&c)
			if s.Err != nil {
				return
			}
			raw, err := DecompressZstd(c)
			if err != nil {
				s.Fail(err)
				return
			}
			var names []string
			in := NewReader(raw, s.ProtoVer)
			in.PackedStrings(&names)
			if in.Err != nil {
				s.Fail(in.Err)
				return
			}
			p.Files = make([]MediaAnnounce, len(names))
			for i, n := range names {
				p.Files[i].Name = n
				s.Fixed(p.Files[i].SHA1[:])
			}
		} else {
			names := make([]string, len(p.Files))
			for i, f := range p.Files {
				names[i] = f.Name
			}
			w := NewWriter(s.ProtoVer)
			w.PackedStrings(&names)
			c := CompressZstd(w.Bytes())
			s.LBlob(&c)
			for i := range p.Files {
				s.Fixed(p.Files[i].SHA1[:])
			}
		}
	} else {
		Slice16(s, &p.Files, func(f *MediaAnnounce) {
			s.Str(&f.Name)
			var b64 string
			if !s.Reading {
				b64 = base64.StdEncoding.EncodeToString(f.SHA1[:])
			}
			s.Str(&b64)
			if s.Reading {
				d, err := base64.StdEncoding.DecodeString(b64)
				if err != nil || len(d) != 20 {
					s.Fail(errors.New("protocol: bad media hash"))
					return
				}
				copy(f.SHA1[:], d)
			}
		})
	}
	s.Str(&p.RemoteURLs)
}

// ToClientItemDef carries the item definitions of the game.
type ToClientItemDef struct {
	Definitions []byte // decompressed serialized ItemDefManager (kept undecoded)
}

func (p *ToClientItemDef) Sync(s *Stream) { s.compressedBlob(&p.Definitions) }

// Sound locations for ToClientPlaySound.Type.
const (
	SoundLocal = iota
	SoundPositional
	SoundObject
)

// ToClientPlaySound starts a sound.
type ToClientPlaySound struct {
	ServerID  int32
	Name      string
	Gain      float32
	Type      uint8 // Sound* constant
	Pos       V3F
	ObjectID  uint16
	Loop      bool
	Fade      float32
	Pitch     float32
	Ephemeral bool
	StartTime float32
}

func (p *ToClientPlaySound) Sync(s *Stream) {
	s.S32(&p.ServerID)
	s.Str(&p.Name)
	s.F32(&p.Gain)
	s.U8(&p.Type)
	s.V3F(&p.Pos)
	s.U16(&p.ObjectID)
	s.Bool(&p.Loop)
	s.F32(&p.Fade)
	s.F32(&p.Pitch)
	if s.More() {
		s.Bool(&p.Ephemeral)
	}
	if s.More() {
		s.F32(&p.StartTime)
	}
}

// ToClientStopSound stops a sound.
type ToClientStopSound struct{ ServerID int32 }

func (p *ToClientStopSound) Sync(s *Stream) { s.S32(&p.ServerID) }

// ToClientPrivileges lists the player's privileges.
type ToClientPrivileges struct{ Privileges []string }

func (p *ToClientPrivileges) Sync(s *Stream) { s.Strings16(&p.Privileges) }

// ToClientInventoryFormspec sets the player's inventory formspec.
type ToClientInventoryFormspec struct{ Formspec string }

func (p *ToClientInventoryFormspec) Sync(s *Stream) { s.LStr(&p.Formspec) }

// ToClientDetachedInventory updates or removes a detached inventory.
type ToClientDetachedInventory struct {
	Name          string
	KeepInventory bool   // false: the inventory is removed
	Inventory     string // Luanti inventory text format (when KeepInventory)
}

func (p *ToClientDetachedInventory) Sync(s *Stream) {
	s.Str(&p.Name)
	s.Bool(&p.KeepInventory)
	if !s.More() {
		return
	}
	n := uint16(len(p.Inventory))
	s.U16(&n) // legacy length field, ignored by clients
	var b []byte
	if !s.Reading {
		b = []byte(p.Inventory)
	}
	s.Rest(&b)
	p.Inventory = string(b)
}

// ToClientShowFormspec opens a formspec.
type ToClientShowFormspec struct {
	Formspec string
	Formname string
}

func (p *ToClientShowFormspec) Sync(s *Stream) { s.LStr(&p.Formspec); s.Str(&p.Formname) }

// ToClientMovement sets the physics parameters.
type ToClientMovement struct {
	AccelerationDefault  float32
	AccelerationAir      float32
	AccelerationFast     float32
	SpeedWalk            float32
	SpeedCrouch          float32
	SpeedFast            float32
	SpeedClimb           float32
	SpeedJump            float32
	LiquidFluidity       float32
	LiquidFluiditySmooth float32
	LiquidSink           float32
	Gravity              float32
}

func (p *ToClientMovement) Sync(s *Stream) {
	for _, f := range []*float32{&p.AccelerationDefault, &p.AccelerationAir, &p.AccelerationFast,
		&p.SpeedWalk, &p.SpeedCrouch, &p.SpeedFast, &p.SpeedClimb, &p.SpeedJump,
		&p.LiquidFluidity, &p.LiquidFluiditySmooth, &p.LiquidSink, &p.Gravity} {
		s.F32(f)
	}
}

// ToClientSpawnParticle spawns a single particle.
type ToClientSpawnParticle struct {
	Params []byte // serialized ParticleParameters (kept undecoded)
}

func (p *ToClientSpawnParticle) Sync(s *Stream) { s.Rest(&p.Params) }

// ToClientAddParticleSpawner creates a particle spawner.
type ToClientAddParticleSpawner struct {
	Amount uint16
	Time   float32
	Params []byte // remaining serialized parameters (kept undecoded)
}

func (p *ToClientAddParticleSpawner) Sync(s *Stream) {
	s.U16(&p.Amount)
	s.F32(&p.Time)
	s.Rest(&p.Params)
}

// ToClientSpawnParticleBatch spawns many particles at once.
type ToClientSpawnParticleBatch struct {
	Particles [][]byte // each a serialized ParticleParameters (kept undecoded)
}

func (p *ToClientSpawnParticleBatch) Sync(s *Stream) {
	if s.Reading {
		var c []byte
		s.LBlob(&c)
		if s.Err != nil {
			return
		}
		raw, err := DecompressZstd(c)
		if err != nil {
			s.Fail(err)
			return
		}
		in := NewReader(raw, s.ProtoVer)
		SliceRest(in, &p.Particles, func(e *[]byte) { in.LBlob(e) })
		s.Fail(in.Err)
		return
	}
	w := NewWriter(s.ProtoVer)
	SliceRest(w, &p.Particles, func(e *[]byte) { w.LBlob(e) })
	c := CompressZstd(w.Bytes())
	s.LBlob(&c)
}

// ToClientCamera restricts the allowed camera modes.
type ToClientCamera struct{ Mode uint8 }

func (p *ToClientCamera) Sync(s *Stream) { s.U8(&p.Mode) }

// ToClientHudAdd creates a HUD element.
type ToClientHudAdd struct {
	ID       uint32
	Type     uint8
	Pos      V2F
	Name     string
	Scale    V2F
	Text     string
	Number   uint32
	Item     uint32
	Dir      uint32
	Align    V2F
	Offset   V2F
	WorldPos V3F
	Size     V2F
	ZIndex   int16
	Text2    string
	Style    uint32
	Flags    uint8
}

func (p *ToClientHudAdd) Sync(s *Stream) {
	s.U32(&p.ID)
	s.U8(&p.Type)
	s.V2F(&p.Pos)
	s.Str(&p.Name)
	s.V2F(&p.Scale)
	s.Str(&p.Text)
	s.U32(&p.Number)
	s.U32(&p.Item)
	s.U32(&p.Dir)
	s.V2F(&p.Align)
	s.V2F(&p.Offset)
	s.V3F(&p.WorldPos)
	s.size(&p.Size)
	p.Flags = 1
	if s.More() {
		s.S16(&p.ZIndex)
		if s.More() {
			s.Str(&p.Text2)
			if s.More() {
				s.U32(&p.Style)
				if s.More() {
					s.U8(&p.Flags)
				}
			}
		}
	}
}

// size reads/writes a HUD size, a v2f since protocol 52 and a v2s32 before.
func (s *Stream) size(p *V2F) {
	if s.ProtoVer >= 52 {
		s.V2F(p)
		return
	}
	var v V2S32
	if !s.Reading {
		v = V2S32{int32(p.X), int32(p.Y)}
	}
	s.V2S32(&v)
	if s.Reading {
		*p = V2F{float32(v.X), float32(v.Y)}
	}
}

// ToClientHudRm removes a HUD element.
type ToClientHudRm struct{ ID uint32 }

func (p *ToClientHudRm) Sync(s *Stream) { s.U32(&p.ID) }

// HUD element stats for ToClientHudChange.Stat.
const (
	HudStatPos = iota
	HudStatName
	HudStatScale
	HudStatText
	HudStatNumber
	HudStatItem
	HudStatDir
	HudStatAlign
	HudStatOffset
	HudStatWorldPos
	HudStatSize
	HudStatZIndex
	HudStatText2
	HudStatStyle
)

// ToClientHudChange changes one property of a HUD element. Depending on Stat,
// exactly one of V2F, String, V3F or Int is meaningful.
type ToClientHudChange struct {
	ID     uint32
	Stat   uint8 // HudStat* constant
	V2F    V2F
	String string
	V3F    V3F
	Int    uint32
}

func (p *ToClientHudChange) Sync(s *Stream) {
	s.U32(&p.ID)
	s.U8(&p.Stat)
	switch p.Stat {
	case HudStatPos, HudStatScale, HudStatAlign, HudStatOffset:
		s.V2F(&p.V2F)
	case HudStatName, HudStatText, HudStatText2:
		s.Str(&p.String)
	case HudStatWorldPos:
		s.V3F(&p.V3F)
	case HudStatSize:
		s.size(&p.V2F)
	default:
		s.U32(&p.Int)
	}
}

// ToClientHudSetFlags changes HUD flags.
type ToClientHudSetFlags struct{ Flags, Mask uint32 }

func (p *ToClientHudSetFlags) Sync(s *Stream) { s.U32(&p.Flags); s.U32(&p.Mask) }

// ToClientHudSetParam sets a HUD parameter (hotbar size or images).
type ToClientHudSetParam struct {
	Param uint16
	Value []byte
}

func (p *ToClientHudSetParam) Sync(s *Stream) { s.U16(&p.Param); s.Blob(&p.Value) }

// ToClientBreath sets the player's breath.
type ToClientBreath struct{ Breath uint16 }

func (p *ToClientBreath) Sync(s *Stream) { s.U16(&p.Breath) }

// SkyColor holds the colors of a "regular" sky.
type SkyColor struct {
	DaySky, DayHorizon, DawnSky, DawnHorizon, NightSky, NightHorizon, Indoors Color
}

// ToClientSetSky configures the sky.
type ToClientSetSky struct {
	BgColor       Color
	Type          string // "regular", "skybox", "plain"
	Clouds        bool
	FogSunTint    Color
	FogMoonTint   Color
	FogTintType   string
	Textures      []string // when Type == "skybox"
	SkyColor      SkyColor // when Type == "regular"
	BodyOrbitTilt float32
	FogDistance   int16
	FogStart      float32
	FogColor      Color
	AutoDimSkybox bool
}

func (p *ToClientSetSky) Sync(s *Stream) {
	s.Color(&p.BgColor)
	s.Str(&p.Type)
	s.Bool(&p.Clouds)
	s.Color(&p.FogSunTint)
	s.Color(&p.FogMoonTint)
	s.Str(&p.FogTintType)
	switch p.Type {
	case "skybox":
		s.Strings16(&p.Textures)
	case "regular":
		c := &p.SkyColor
		for _, col := range []*Color{&c.DaySky, &c.DayHorizon, &c.DawnSky, &c.DawnHorizon,
			&c.NightSky, &c.NightHorizon, &c.Indoors} {
			s.Color(col)
		}
	}
	p.AutoDimSkybox = true
	if !s.More() {
		return
	}
	s.F32(&p.BodyOrbitTilt)
	if !s.More() {
		return
	}
	s.S16(&p.FogDistance)
	s.F32(&p.FogStart)
	if !s.More() {
		return
	}
	s.Color(&p.FogColor)
	if !s.More() {
		return
	}
	s.Bool(&p.AutoDimSkybox)
}

// ToClientOverrideDayNightRatio overrides the day/night ratio.
type ToClientOverrideDayNightRatio struct {
	Override bool
	Ratio    uint16 // 0..65535
}

func (p *ToClientOverrideDayNightRatio) Sync(s *Stream) { s.Bool(&p.Override); s.U16(&p.Ratio) }

// ToClientLocalPlayerAnimations sets the player model animation frames.
type ToClientLocalPlayerAnimations struct {
	Idle, Walk, Dig, WalkDig V2F
	FrameSpeed               float32
}

func (p *ToClientLocalPlayerAnimations) Sync(s *Stream) {
	for _, v := range []*V2F{&p.Idle, &p.Walk, &p.Dig, &p.WalkDig} {
		if s.ProtoVer >= 46 {
			s.V2F(v)
		} else {
			s.size(v)
		}
	}
	s.F32(&p.FrameSpeed)
}

// ToClientEyeOffset sets the camera offsets.
type ToClientEyeOffset struct{ First, Third, ThirdFront V3F }

func (p *ToClientEyeOffset) Sync(s *Stream) {
	s.V3F(&p.First)
	s.V3F(&p.Third)
	p.ThirdFront = p.Third
	if s.More() {
		s.V3F(&p.ThirdFront)
	}
}

// ToClientDeleteParticleSpawner removes a particle spawner.
type ToClientDeleteParticleSpawner struct{ ID uint32 }

func (p *ToClientDeleteParticleSpawner) Sync(s *Stream) { s.U32(&p.ID) }

// ToClientCloudParams configures clouds.
type ToClientCloudParams struct {
	Density      float32
	ColorBright  Color
	ColorAmbient Color
	Height       float32
	Thickness    float32
	Speed        V2F
	ColorShadow  Color
}

func (p *ToClientCloudParams) Sync(s *Stream) {
	s.F32(&p.Density)
	s.Color(&p.ColorBright)
	s.Color(&p.ColorAmbient)
	s.F32(&p.Height)
	s.F32(&p.Thickness)
	s.V2F(&p.Speed)
	p.ColorShadow = 0xffcccccc
	if s.More() {
		s.Color(&p.ColorShadow)
	}
}

// ToClientFadeSound fades a sound.
type ToClientFadeSound struct {
	ServerID int32
	Step     float32
	Gain     float32
}

func (p *ToClientFadeSound) Sync(s *Stream) { s.S32(&p.ServerID); s.F32(&p.Step); s.F32(&p.Gain) }

// Player list update types for ToClientUpdatePlayerList.Type.
const (
	PlayerListInit = iota
	PlayerListAdd
	PlayerListRemove
)

// ToClientUpdatePlayerList informs about players that are online.
type ToClientUpdatePlayerList struct {
	Type    uint8 // PlayerList* constant
	Players []string
}

func (p *ToClientUpdatePlayerList) Sync(s *Stream) { s.U8(&p.Type); s.Strings16(&p.Players) }

// ToClientModChannelMsg delivers a mod channel message.
type ToClientModChannelMsg struct{ Channel, Sender, Message string }

func (p *ToClientModChannelMsg) Sync(s *Stream) {
	s.Str(&p.Channel)
	s.Str(&p.Sender)
	s.Str(&p.Message)
}

// Signals for ToClientModChannelSignal.Signal.
const (
	ModChannelJoinOK = iota
	ModChannelJoinFailure
	ModChannelLeaveOK
	ModChannelLeaveFailure
	ModChannelNotRegistered
	ModChannelSetState
)

// ToClientModChannelSignal reports a mod channel state change.
type ToClientModChannelSignal struct {
	Signal  uint8 // ModChannel* constant
	Channel string
	State   uint8 // only for ModChannelSetState
}

func (p *ToClientModChannelSignal) Sync(s *Stream) {
	s.U8(&p.Signal)
	s.Str(&p.Channel)
	if p.Signal == ModChannelSetState && s.More() {
		s.U8(&p.State)
	}
}

// ToClientNodemetaChanged carries changed node metadata.
type ToClientNodemetaChanged struct {
	Data []byte // zlib-compressed serialized metadata list (kept undecoded)
}

func (p *ToClientNodemetaChanged) Sync(s *Stream) { s.LBlob(&p.Data) }

// ToClientSetSun configures the sun.
type ToClientSetSun struct {
	Visible        bool
	Texture        string
	Tonemap        string
	Sunrise        string
	SunriseVisible bool
	Scale          float32
}

func (p *ToClientSetSun) Sync(s *Stream) {
	s.Bool(&p.Visible)
	s.Str(&p.Texture)
	s.Str(&p.Tonemap)
	s.Str(&p.Sunrise)
	s.Bool(&p.SunriseVisible)
	s.F32(&p.Scale)
}

// ToClientSetMoon configures the moon.
type ToClientSetMoon struct {
	Visible bool
	Texture string
	Tonemap string
	Scale   float32
}

func (p *ToClientSetMoon) Sync(s *Stream) {
	s.Bool(&p.Visible)
	s.Str(&p.Texture)
	s.Str(&p.Tonemap)
	s.F32(&p.Scale)
}

// ToClientSetStars configures the stars.
type ToClientSetStars struct {
	Visible    bool
	Count      uint32
	Color      Color
	Scale      float32
	DayOpacity float32
	StarSeed   uint64
}

func (p *ToClientSetStars) Sync(s *Stream) {
	s.Bool(&p.Visible)
	s.U32(&p.Count)
	s.Color(&p.Color)
	s.F32(&p.Scale)
	if s.More() {
		s.F32(&p.DayOpacity)
	}
	if s.More() {
		s.U64(&p.StarSeed)
	}
}

// ToClientMovePlayerRel moves the player relative to its position.
type ToClientMovePlayerRel struct{ AddedPos V3F }

func (p *ToClientMovePlayerRel) Sync(s *Stream) { s.V3F(&p.AddedPos) }

// ToClientSRPBytesSB is the server's SRP challenge.
type ToClientSRPBytesSB struct {
	Salt []byte
	B    []byte
}

func (p *ToClientSRPBytesSB) Sync(s *Stream) { s.Blob(&p.Salt); s.Blob(&p.B) }

// ToClientFormspecPrepend sets the formspec prepend string.
type ToClientFormspecPrepend struct{ Formspec string }

func (p *ToClientFormspecPrepend) Sync(s *Stream) { s.Str(&p.Formspec) }

// MinimapMode describes one minimap mode.
type MinimapMode struct {
	Type    uint16
	Label   string
	Size    uint16
	Texture string
	Scale   uint16
}

// ToClientMinimapModes defines the available minimap modes.
type ToClientMinimapModes struct {
	Mode  uint16 // index of the mode to select
	Modes []MinimapMode
}

func (p *ToClientMinimapModes) Sync(s *Stream) {
	// wire order: count, selected mode, then the modes
	n := uint16(len(p.Modes))
	s.U16(&n)
	s.U16(&p.Mode)
	if s.Reading {
		var ok bool
		var cnt int
		if cnt, ok = s.readCount(int(n)); !ok {
			return
		}
		p.Modes = make([]MinimapMode, cnt)
	}
	for i := range p.Modes {
		m := &p.Modes[i]
		s.U16(&m.Type)
		s.Str(&m.Label)
		s.U16(&m.Size)
		s.Str(&m.Texture)
		s.U16(&m.Scale)
	}
}

// ToClientSetLighting configures lighting and post-processing.
type ToClientSetLighting struct {
	ShadowIntensity         float32
	Saturation              float32
	LuminanceMin            float32
	LuminanceMax            float32
	ExposureCorrection      float32
	SpeedDarkBright         float32
	SpeedBrightDark         float32
	CenterWeightPower       float32
	VolumetricLightStrength float32
	ShadowTint              Color
	BloomIntensity          float32
	BloomStrengthFactor     float32
	BloomRadius             float32
	ShadowDirection         V3F
	LightCurveSet           bool
	LightCurve              [16]byte
	AOGamma                 float32
}

func (p *ToClientSetLighting) Sync(s *Stream) {
	s.F32(&p.ShadowIntensity)
	if !s.More() {
		return
	}
	for _, f := range []*float32{&p.Saturation, &p.LuminanceMin, &p.LuminanceMax, &p.ExposureCorrection,
		&p.SpeedDarkBright, &p.SpeedBrightDark, &p.CenterWeightPower} {
		s.F32(f)
	}
	if !s.More() {
		return
	}
	s.F32(&p.VolumetricLightStrength)
	if !s.More() {
		return
	}
	s.Color(&p.ShadowTint)
	s.F32(&p.BloomIntensity)
	s.F32(&p.BloomStrengthFactor)
	s.F32(&p.BloomRadius)
	if !s.More() {
		return
	}
	s.V3F(&p.ShadowDirection)
	if !s.More() {
		return
	}
	s.Bool(&p.LightCurveSet)
	if p.LightCurveSet {
		s.Fixed(p.LightCurve[:])
	}
	s.F32(&p.AOGamma)
}
