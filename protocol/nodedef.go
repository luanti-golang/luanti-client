package protocol

import (
	"errors"
	"fmt"
	"sort"
)

// NodeDefinition is the decoded definition of one node type, as sent in
// ToClientNodeDef. Enumerations are exposed by name; lengths and positions
// (node boxes) are in BS units, 10 per node.
type NodeDefinition struct {
	ID     uint16           `json:"id"`
	Name   string           `json:"name"`
	Groups map[string]int16 `json:"groups,omitempty"`

	ParamType  string `json:"param_type"`
	ParamType2 string `json:"param_type2"`

	DrawType         string     `json:"drawtype"`
	Mesh             string     `json:"mesh,omitempty"`
	VisualScale      float32    `json:"visual_scale"`
	Tiles            [6]TileDef `json:"tiles"`
	OverlayTiles     [6]TileDef `json:"overlay_tiles"`
	SpecialTiles     [6]TileDef `json:"special_tiles"`
	Alpha            string     `json:"alpha"`
	Color            RGB        `json:"color"`
	PaletteName      string     `json:"palette,omitempty"`
	Waving           uint8      `json:"waving"`
	ConnectSides     uint8      `json:"connect_sides"`
	ConnectsToID     []uint16   `json:"connects_to_ids,omitempty"`
	PostEffect       Color      `json:"post_effect_color"`
	PostEffectShaded bool       `json:"post_effect_color_shaded"`
	Leveled          uint8      `json:"leveled"`
	LeveledMax       uint8      `json:"leveled_max"`

	LightPropagates    bool  `json:"light_propagates"`
	SunlightPropagates bool  `json:"sunlight_propagates"`
	LightSource        uint8 `json:"light_source"`

	IsGroundContent bool   `json:"is_ground_content"`
	Walkable        bool   `json:"walkable"`
	Pointable       string `json:"pointable"` // "true", "false" or "blocking"
	Diggable        bool   `json:"diggable"`
	Climbable       bool   `json:"climbable"`
	BuildableTo     bool   `json:"buildable_to"`
	Rightclickable  bool   `json:"rightclickable"`
	DamagePerSecond uint32 `json:"damage_per_second"`

	LiquidType               string `json:"liquid_type"`
	LiquidAlternativeFlowing string `json:"liquid_alternative_flowing,omitempty"`
	LiquidAlternativeSource  string `json:"liquid_alternative_source,omitempty"`
	LiquidViscosity          uint8  `json:"liquid_viscosity"`
	LiquidRenewable          bool   `json:"liquid_renewable"`
	LiquidRange              uint8  `json:"liquid_range"`
	LiquidMovePhysics        bool   `json:"liquid_move_physics"`
	Drowning                 uint8  `json:"drowning"`
	Floodable                bool   `json:"floodable"`
	MoveResistance           uint8  `json:"move_resistance"`

	NodeBox      NodeBox `json:"node_box"`
	SelectionBox NodeBox `json:"selection_box"`
	CollisionBox NodeBox `json:"collision_box"`

	SoundFootstep SoundSpec `json:"sound_footstep"`
	SoundDig      SoundSpec `json:"sound_dig"`
	SoundDug      SoundSpec `json:"sound_dug"`

	LegacyFacedirSimple bool   `json:"legacy_facedir_simple"`
	LegacyWallmounted   bool   `json:"legacy_wallmounted"`
	NodeDigPrediction   string `json:"node_dig_prediction"`
}

// RGB is a 24-bit color.
type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// TileDef describes one tile (face texture) of a node.
type TileDef struct {
	Name               string         `json:"name"`
	Animation          *TileAnimation `json:"animation,omitempty"`
	BackfaceCulling    bool           `json:"backface_culling"`
	TileableHorizontal bool           `json:"tileable_horizontal"`
	TileableVertical   bool           `json:"tileable_vertical"`
	Color              *RGB           `json:"color,omitempty"`
	Scale              uint8          `json:"scale,omitempty"`
	AlignStyle         string         `json:"align_style"`
}

// TileAnimation is an animated texture description.
type TileAnimation struct {
	Type string `json:"type"` // "vertical_frames" or "sheet_2d"
	// vertical_frames
	AspectW uint16  `json:"aspect_w,omitempty"`
	AspectH uint16  `json:"aspect_h,omitempty"`
	Length  float32 `json:"length,omitempty"`
	// sheet_2d
	FramesW     uint8   `json:"frames_w,omitempty"`
	FramesH     uint8   `json:"frames_h,omitempty"`
	FrameLength float32 `json:"frame_length,omitempty"`
}

// SoundSpec is a simple sound description.
type SoundSpec struct {
	Name  string  `json:"name"`
	Gain  float32 `json:"gain"`
	Pitch float32 `json:"pitch"`
	Fade  float32 `json:"fade"`
}

// AABB is an axis-aligned box.
type AABB struct {
	Min V3F `json:"min"`
	Max V3F `json:"max"`
}

// NodeBox is a node box, selection box or collision box.
type NodeBox struct {
	Type       string        `json:"type"` // regular, fixed, wallmounted, leveled, connected
	Fixed      []AABB        `json:"fixed,omitempty"`
	WallTop    *AABB         `json:"wall_top,omitempty"`
	WallBottom *AABB         `json:"wall_bottom,omitempty"`
	WallSide   *AABB         `json:"wall_side,omitempty"`
	Connected  *ConnectedBox `json:"connected,omitempty"`
}

// ConnectedBox holds the extra boxes of a "connected" node box.
type ConnectedBox struct {
	ConnectTop         []AABB `json:"connect_top,omitempty"`
	ConnectBottom      []AABB `json:"connect_bottom,omitempty"`
	ConnectFront       []AABB `json:"connect_front,omitempty"`
	ConnectLeft        []AABB `json:"connect_left,omitempty"`
	ConnectBack        []AABB `json:"connect_back,omitempty"`
	ConnectRight       []AABB `json:"connect_right,omitempty"`
	DisconnectedTop    []AABB `json:"disconnected_top,omitempty"`
	DisconnectedBottom []AABB `json:"disconnected_bottom,omitempty"`
	DisconnectedFront  []AABB `json:"disconnected_front,omitempty"`
	DisconnectedLeft   []AABB `json:"disconnected_left,omitempty"`
	DisconnectedBack   []AABB `json:"disconnected_back,omitempty"`
	DisconnectedRight  []AABB `json:"disconnected_right,omitempty"`
	Disconnected       []AABB `json:"disconnected,omitempty"`
	DisconnectedSides  []AABB `json:"disconnected_sides,omitempty"`
}

var (
	paramTypeNames  = []string{"none", "light"}
	paramType2Names = []string{"none", "full", "flowingliquid", "facedir", "wallmounted", "leveled",
		"degrotate", "meshoptions", "color", "colorfacedir", "colorwallmounted", "glasslikeliquidlevel",
		"colordegrotate", "4dir", "color4dir"}
	drawTypeNames = []string{"normal", "airlike", "liquid", "flowingliquid", "glasslike", "allfaces",
		"allfaces_optional", "torchlike", "signlike", "plantlike", "fencelike", "raillike", "nodebox",
		"glasslike_framed", "firelike", "glasslike_framed_optional", "mesh", "plantlike_rooted"}
	liquidTypeNames = []string{"none", "flowing", "source"}
	alphaNames      = []string{"blend", "clip", "opaque", "opaque"}
	boxTypeNames    = []string{"regular", "fixed", "wallmounted", "leveled", "connected"}
	alignNames      = []string{"node", "world", "user"}
	pointableNames  = []string{"false", "true", "blocking"}
)

func enumName(names []string, v uint8) string {
	if int(v) < len(names) {
		return names[v]
	}
	return fmt.Sprintf("unknown(%d)", v)
}

// ParseNodeDefs decodes the serialized NodeDefManager carried by
// ToClientNodeDef.Definitions. The result is sorted by node id. Luanti's
// built-in nodes (air, unknown, ignore) are not part of the data.
func ParseNodeDefs(data []byte, protoVer uint16) ([]NodeDefinition, error) {
	if protoVer == 0 {
		protoVer = LatestProtocolVersion
	}
	s := NewReader(data, protoVer)
	var version uint8
	var count uint16
	s.U8(&version)
	s.U16(&count)
	var body []byte
	s.LBlob(&body)
	if s.Err != nil {
		return nil, s.Err
	}
	if version < 1 {
		return nil, errors.New("protocol: unsupported node definition version")
	}
	in := NewReader(body, protoVer)
	defs := make([]NodeDefinition, 0, count)
	for i := 0; i < int(count); i++ {
		var id uint16
		var wrapped []byte
		in.U16(&id)
		in.Blob(&wrapped)
		if in.Err != nil {
			return nil, fmt.Errorf("node %d of %d: %w", i, count, in.Err)
		}
		d, err := parseContentFeatures(wrapped, protoVer)
		if err != nil {
			return nil, fmt.Errorf("node id %d: %w", id, err)
		}
		d.ID = id
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs, nil
}

// NodeDefs decodes the node definitions of the packet.
func (p *ToClientNodeDef) NodeDefs(protoVer uint16) ([]NodeDefinition, error) {
	return ParseNodeDefs(p.Definitions, protoVer)
}

func parseContentFeatures(b []byte, protoVer uint16) (NodeDefinition, error) {
	var d NodeDefinition
	s := NewReader(b, protoVer)
	var version, u8 uint8
	s.U8(&version)
	if s.Err == nil && version < 13 {
		return d, fmt.Errorf("unsupported ContentFeatures version %d", version)
	}
	s.Str(&d.Name)
	var ngroups uint16
	s.U16(&ngroups)
	if ngroups > 0 && s.Err == nil {
		d.Groups = map[string]int16{}
		for i := 0; i < int(ngroups) && s.Err == nil; i++ {
			var name string
			var v int16
			s.Str(&name)
			s.S16(&v)
			d.Groups[name] = v
		}
	}
	s.U8(&u8)
	d.ParamType = enumName(paramTypeNames, u8)
	s.U8(&u8)
	d.ParamType2 = enumName(paramType2Names, u8)

	s.U8(&u8)
	d.DrawType = enumName(drawTypeNames, u8)
	s.Str(&d.Mesh)
	s.F32(&d.VisualScale)
	var tiles uint8
	s.U8(&tiles)
	if s.Err == nil && tiles != 6 {
		return d, fmt.Errorf("unsupported tile count %d", tiles)
	}
	for i := range d.Tiles {
		d.Tiles[i] = parseTile(s)
	}
	for i := range d.OverlayTiles {
		d.OverlayTiles[i] = parseTile(s)
	}
	var special uint8
	s.U8(&special)
	if s.Err == nil && special != 6 {
		return d, fmt.Errorf("unsupported special tile count %d", special)
	}
	for i := range d.SpecialTiles {
		d.SpecialTiles[i] = parseTile(s)
	}
	var legacyAlpha uint8
	s.U8(&legacyAlpha)
	d.Alpha = "opaque"
	if legacyAlpha != 255 {
		d.Alpha = "blend"
	}
	s.U8(&d.Color.R)
	s.U8(&d.Color.G)
	s.U8(&d.Color.B)
	s.Str(&d.PaletteName)
	s.U8(&d.Waving)
	s.U8(&d.ConnectSides)
	Slice16(s, &d.ConnectsToID, func(e *uint16) { s.U16(e) })
	s.Color(&d.PostEffect)
	s.U8(&d.Leveled)

	s.Bool(&d.LightPropagates)
	s.Bool(&d.SunlightPropagates)
	s.U8(&d.LightSource)
	s.Bool(&d.IsGroundContent)

	s.Bool(&d.Walkable)
	s.U8(&u8)
	d.Pointable = enumName(pointableNames, u8)
	s.Bool(&d.Diggable)
	s.Bool(&d.Climbable)
	s.Bool(&d.BuildableTo)
	s.Bool(&d.Rightclickable)
	s.U32(&d.DamagePerSecond)

	s.U8(&u8)
	d.LiquidType = enumName(liquidTypeNames, u8)
	d.LiquidMovePhysics = u8 != 0
	s.Str(&d.LiquidAlternativeFlowing)
	s.Str(&d.LiquidAlternativeSource)
	s.U8(&d.LiquidViscosity)
	d.MoveResistance = d.LiquidViscosity
	s.Bool(&d.LiquidRenewable)
	s.U8(&d.LiquidRange)
	s.U8(&d.Drowning)
	s.Bool(&d.Floodable)

	d.NodeBox = parseNodeBox(s)
	d.SelectionBox = parseNodeBox(s)
	d.CollisionBox = parseNodeBox(s)

	d.SoundFootstep = parseSound(s)
	d.SoundDig = parseSound(s)
	d.SoundDug = parseSound(s)

	s.Bool(&d.LegacyFacedirSimple)
	s.Bool(&d.LegacyWallmounted)

	// Fields added in later versions are optional.
	s.Str(&d.NodeDigPrediction)
	if s.More() {
		s.U8(&d.LeveledMax)
		if s.More() {
			s.U8(&u8)
			d.Alpha = enumName(alphaNames, u8)
			if s.More() {
				s.U8(&d.MoveResistance)
				s.Bool(&d.LiquidMovePhysics)
				if s.More() {
					s.Bool(&d.PostEffectShaded)
				}
			}
		}
	}
	return d, s.Err
}

func parseTile(s *Stream) TileDef {
	var t TileDef
	var ver, kind uint8
	s.U8(&ver)
	if s.Err == nil && ver < 6 {
		s.Fail(fmt.Errorf("unsupported TileDef version %d", ver))
		return t
	}
	s.Str(&t.Name)
	s.U8(&kind)
	switch kind {
	case 1:
		a := &TileAnimation{Type: "vertical_frames"}
		s.U16(&a.AspectW)
		s.U16(&a.AspectH)
		s.F32(&a.Length)
		t.Animation = a
	case 2:
		a := &TileAnimation{Type: "sheet_2d"}
		s.U8(&a.FramesW)
		s.U8(&a.FramesH)
		s.F32(&a.FrameLength)
		t.Animation = a
	}
	var flags uint16
	s.U16(&flags)
	t.BackfaceCulling = flags&(1<<0) != 0
	t.TileableHorizontal = flags&(1<<1) != 0
	t.TileableVertical = flags&(1<<2) != 0
	if flags&(1<<3) != 0 {
		c := &RGB{}
		s.U8(&c.R)
		s.U8(&c.G)
		s.U8(&c.B)
		t.Color = c
	}
	if flags&(1<<4) != 0 {
		s.U8(&t.Scale)
	}
	align := uint8(0)
	if flags&(1<<5) != 0 {
		s.U8(&align)
	}
	t.AlignStyle = enumName(alignNames, align)
	return t
}

func parseSound(s *Stream) SoundSpec {
	var v SoundSpec
	s.Str(&v.Name)
	s.F32(&v.Gain)
	s.F32(&v.Pitch)
	s.F32(&v.Fade)
	return v
}

func parseBoxes(s *Stream) []AABB {
	var n uint16
	s.U16(&n)
	var out []AABB
	if c, ok := s.readCount(int(n)); ok {
		out = make([]AABB, 0, c)
		for i := 0; i < c && s.Err == nil; i++ {
			var b AABB
			s.V3F(&b.Min)
			s.V3F(&b.Max)
			out = append(out, b)
		}
	}
	return out
}

func parseBox(s *Stream) *AABB {
	var b AABB
	s.V3F(&b.Min)
	s.V3F(&b.Max)
	return &b
}

func parseNodeBox(s *Stream) NodeBox {
	var nb NodeBox
	var ver, kind uint8
	s.U8(&ver)
	if s.Err == nil && ver < 6 {
		s.Fail(fmt.Errorf("unsupported NodeBox version %d", ver))
		return nb
	}
	s.U8(&kind)
	nb.Type = enumName(boxTypeNames, kind)
	switch kind {
	case 1, 3:
		nb.Fixed = parseBoxes(s)
	case 2:
		nb.WallTop, nb.WallBottom, nb.WallSide = parseBox(s), parseBox(s), parseBox(s)
	case 4:
		nb.Fixed = parseBoxes(s)
		c := &ConnectedBox{}
		for _, dst := range []*[]AABB{&c.ConnectTop, &c.ConnectBottom, &c.ConnectFront, &c.ConnectLeft,
			&c.ConnectBack, &c.ConnectRight, &c.DisconnectedTop, &c.DisconnectedBottom,
			&c.DisconnectedFront, &c.DisconnectedLeft, &c.DisconnectedBack, &c.DisconnectedRight,
			&c.Disconnected, &c.DisconnectedSides} {
			*dst = parseBoxes(s)
		}
		nb.Connected = c
	}
	return nb
}
