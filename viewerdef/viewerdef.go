// Package viewerdef converts node definitions received from a server into the
// JSON schema of luanti-viewer (NodeDefinition in its src/types.ts), which
// follows the shape of the Lua register_node API.
//
// The protocol form is already resolved by the server (6 tiles, alpha mode,
// per-drawtype backface culling defaults, connects_to groups expanded), so the
// conversion mostly renames fields, converts node boxes from BS units to node
// units and replaces content ids with node names. Data the viewer does not
// render (sounds, selection/collision boxes, overlay tiles, palettes, ...) is
// dropped.
package viewerdef

import (
	"fmt"
	"math"

	"github.com/luanti-golang/luanti-client/protocol"
)

// bs is the number of protocol length units (BS) per node.
const bs = 10

// NodeDefinition mirrors NodeDefinition of luanti-viewer.
type NodeDefinition struct {
	Name        string           `json:"name"`
	DrawType    string           `json:"drawtype"`
	ParamType   string           `json:"paramtype"`
	ParamType2  string           `json:"paramtype2"`
	LightSource uint8            `json:"light_source"`
	Walkable    bool             `json:"walkable"`
	Groups      map[string]int16 `json:"groups"` // null if the node has no groups

	Tiles           []Tile   `json:"tiles,omitempty"`
	SpecialTiles    []Tile   `json:"special_tiles,omitempty"`
	UseTextureAlpha string   `json:"use_texture_alpha,omitempty"`
	VisualScale     float64  `json:"visual_scale"`
	Color           string   `json:"color,omitempty"`
	NodeBox         *NodeBox `json:"node_box,omitempty"`
	Mesh            string   `json:"mesh,omitempty"`
	ConnectsTo      []string `json:"connects_to,omitempty"`

	LiquidAlternativeSource  string `json:"liquid_alternative_source,omitempty"`
	LiquidAlternativeFlowing string `json:"liquid_alternative_flowing,omitempty"`
	LiquidRange              uint8  `json:"liquid_range"`
	PostEffectColor          *Color `json:"post_effect_color,omitempty"`
}

// Tile mirrors TileDefinition of luanti-viewer.
type Tile struct {
	Name               string     `json:"name"`
	BackfaceCulling    bool       `json:"backface_culling"`
	TileableVertical   bool       `json:"tileable_vertical,omitempty"`
	TileableHorizontal bool       `json:"tileable_horizontal,omitempty"`
	AlignStyle         string     `json:"align_style,omitempty"` // omitted for "node"
	Scale              uint8      `json:"scale,omitempty"`
	Animation          *Animation `json:"animation,omitempty"`
}

// Animation mirrors TileAnimation of luanti-viewer.
type Animation struct {
	Type string `json:"type"`
	// vertical_frames
	AspectW *uint16  `json:"aspect_w,omitempty"`
	AspectH *uint16  `json:"aspect_h,omitempty"`
	Length  *float32 `json:"length,omitempty"`
	// sheet_2d
	FramesW     *uint8   `json:"frames_w,omitempty"`
	FramesH     *uint8   `json:"frames_h,omitempty"`
	FrameLength *float32 `json:"frame_length,omitempty"`
}

// Box is [x1, y1, z1, x2, y2, z2] in node units.
type Box [6]float64

// NodeBox mirrors NodeBox of luanti-viewer.
type NodeBox struct {
	Type  string `json:"type"`
	Fixed []Box  `json:"fixed,omitempty"`

	ConnectTop         []Box `json:"connect_top,omitempty"`
	ConnectBottom      []Box `json:"connect_bottom,omitempty"`
	ConnectFront       []Box `json:"connect_front,omitempty"`
	ConnectBack        []Box `json:"connect_back,omitempty"`
	ConnectLeft        []Box `json:"connect_left,omitempty"`
	ConnectRight       []Box `json:"connect_right,omitempty"`
	DisconnectedTop    []Box `json:"disconnected_top,omitempty"`
	DisconnectedBottom []Box `json:"disconnected_bottom,omitempty"`
	DisconnectedFront  []Box `json:"disconnected_front,omitempty"`
	DisconnectedBack   []Box `json:"disconnected_back,omitempty"`
	DisconnectedLeft   []Box `json:"disconnected_left,omitempty"`
	DisconnectedRight  []Box `json:"disconnected_right,omitempty"`
	Disconnected       []Box `json:"disconnected,omitempty"`
	DisconnectedSides  []Box `json:"disconnected_sides,omitempty"`

	WallTop    *Box `json:"wall_top,omitempty"`
	WallBottom *Box `json:"wall_bottom,omitempty"`
	WallSide   *Box `json:"wall_side,omitempty"`
}

// Color mirrors Color of luanti-viewer.
type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

var (
	drawTypes = set("normal", "airlike", "liquid", "flowingliquid", "glasslike", "glasslike_framed",
		"glasslike_framed_optional", "allfaces", "allfaces_optional", "torchlike", "signlike", "plantlike",
		"firelike", "fencelike", "raillike", "nodebox", "mesh", "plantlike_rooted")
	paramTypes2 = set("none", "facedir", "4dir", "wallmounted", "meshoptions", "flowingliquid", "leveled",
		"degrotate", "glasslikeliquidlevel", "color", "colorfacedir", "color4dir", "colorwallmounted",
		"colordegrotate")
	alphaModes = set("opaque", "clip", "blend")
	boxTypes   = set("regular", "fixed", "leveled", "connected", "wallmounted")
)

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// Air is the definition of Luanti's built-in "air" node, which servers do not
// send.
var Air = NodeDefinition{
	Name:        "air",
	DrawType:    "airlike",
	ParamType:   "light",
	ParamType2:  "none",
	Groups:      map[string]int16{"not_in_creative_inventory": 1},
	VisualScale: 1,
	LiquidRange: 8,
}

// Convert converts the node definitions of a server into the viewer schema,
// keyed by node name (the format of the viewer's nodedefs.json). The built-in
// "air" node is added unless the server defines it.
func Convert(defs []protocol.NodeDefinition) map[string]NodeDefinition {
	names := make(map[uint16]string, len(defs))
	for i := range defs {
		names[defs[i].ID] = defs[i].Name
	}
	out := make(map[string]NodeDefinition, len(defs)+1)
	out[Air.Name] = Air
	for i := range defs {
		d := ConvertOne(&defs[i], names)
		out[d.Name] = d
	}
	return out
}

// ConvertOne converts a single node definition. names maps content ids to node
// names and resolves connects_to; unknown ids are skipped.
func ConvertOne(d *protocol.NodeDefinition, names map[uint16]string) NodeDefinition {
	v := NodeDefinition{
		Name:                     d.Name,
		DrawType:                 d.DrawType,
		ParamType:                d.ParamType,
		ParamType2:               d.ParamType2,
		LightSource:              d.LightSource,
		Walkable:                 d.Walkable,
		VisualScale:              round(float64(d.VisualScale)),
		LiquidAlternativeSource:  d.LiquidAlternativeSource,
		LiquidAlternativeFlowing: d.LiquidAlternativeFlowing,
		LiquidRange:              d.LiquidRange,
	}
	if len(d.Groups) > 0 {
		v.Groups = d.Groups
	}
	if !drawTypes[v.DrawType] {
		v.DrawType = "normal"
	}
	if v.ParamType != "light" {
		v.ParamType = "none"
	}
	// "full" (param2 without a special meaning) and unknown values
	if !paramTypes2[v.ParamType2] {
		v.ParamType2 = "none"
	}
	if alphaModes[d.Alpha] {
		v.UseTextureAlpha = d.Alpha
	}
	if d.Color != (protocol.RGB{R: 255, G: 255, B: 255}) {
		v.Color = fmt.Sprintf("#%02x%02x%02x", d.Color.R, d.Color.G, d.Color.B)
	}
	if a := uint8(d.PostEffect >> 24); a != 0 {
		v.PostEffectColor = &Color{R: uint8(d.PostEffect >> 16), G: uint8(d.PostEffect >> 8), B: uint8(d.PostEffect), A: a}
	}

	if v.DrawType != "airlike" {
		v.Tiles = convertTiles(d.Tiles[:])
	}
	v.SpecialTiles = convertTiles(d.SpecialTiles[:])

	switch v.DrawType {
	case "nodebox":
		v.NodeBox = convertNodeBox(d.NodeBox)
	case "mesh":
		v.Mesh = d.Mesh
	}
	for _, id := range d.ConnectsToID {
		if n, ok := names[id]; ok {
			v.ConnectsTo = append(v.ConnectsTo, n)
		}
	}
	return v
}

// convertTiles drops the trailing tiles without a texture, nil if none has one.
func convertTiles(tiles []protocol.TileDef) []Tile {
	n := len(tiles)
	for n > 0 && tiles[n-1].Name == "" {
		n--
	}
	if n == 0 {
		return nil
	}
	out := make([]Tile, n)
	for i, t := range tiles[:n] {
		out[i] = Tile{
			Name:               t.Name,
			BackfaceCulling:    t.BackfaceCulling,
			TileableVertical:   t.TileableVertical,
			TileableHorizontal: t.TileableHorizontal,
			Scale:              t.Scale,
		}
		if t.AlignStyle == "world" || t.AlignStyle == "user" {
			out[i].AlignStyle = t.AlignStyle
		}
		if a := t.Animation; a != nil {
			switch a.Type {
			case "vertical_frames":
				out[i].Animation = &Animation{Type: a.Type, AspectW: &a.AspectW, AspectH: &a.AspectH, Length: &a.Length}
			case "sheet_2d":
				out[i].Animation = &Animation{Type: a.Type, FramesW: &a.FramesW, FramesH: &a.FramesH, FrameLength: &a.FrameLength}
			}
		}
	}
	return out
}

func convertNodeBox(nb protocol.NodeBox) *NodeBox {
	v := &NodeBox{Type: nb.Type}
	switch nb.Type {
	case "fixed", "leveled":
		v.Fixed = boxes(nb.Fixed)
		if v.Fixed == nil {
			v.Fixed = []Box{}
		}
	case "connected":
		v.Fixed = boxes(nb.Fixed)
		if c := nb.Connected; c != nil {
			v.ConnectTop = boxes(c.ConnectTop)
			v.ConnectBottom = boxes(c.ConnectBottom)
			v.ConnectFront = boxes(c.ConnectFront)
			v.ConnectBack = boxes(c.ConnectBack)
			v.ConnectLeft = boxes(c.ConnectLeft)
			v.ConnectRight = boxes(c.ConnectRight)
			v.DisconnectedTop = boxes(c.DisconnectedTop)
			v.DisconnectedBottom = boxes(c.DisconnectedBottom)
			v.DisconnectedFront = boxes(c.DisconnectedFront)
			v.DisconnectedBack = boxes(c.DisconnectedBack)
			v.DisconnectedLeft = boxes(c.DisconnectedLeft)
			v.DisconnectedRight = boxes(c.DisconnectedRight)
			v.Disconnected = boxes(c.Disconnected)
			v.DisconnectedSides = boxes(c.DisconnectedSides)
		}
	case "wallmounted":
		v.WallTop = boxPtr(nb.WallTop)
		v.WallBottom = boxPtr(nb.WallBottom)
		v.WallSide = boxPtr(nb.WallSide)
		if v.WallTop == nil || v.WallBottom == nil || v.WallSide == nil {
			return &NodeBox{Type: "regular"}
		}
	}
	if !boxTypes[v.Type] {
		return &NodeBox{Type: "regular"}
	}
	return v
}

func boxes(in []protocol.AABB) []Box {
	if len(in) == 0 {
		return nil
	}
	out := make([]Box, len(in))
	for i, b := range in {
		out[i] = box(b)
	}
	return out
}

func boxPtr(b *protocol.AABB) *Box {
	if b == nil {
		return nil
	}
	v := box(*b)
	return &v
}

func box(b protocol.AABB) Box {
	return Box{
		round(float64(b.Min.X) / bs), round(float64(b.Min.Y) / bs), round(float64(b.Min.Z) / bs),
		round(float64(b.Max.X) / bs), round(float64(b.Max.Y) / bs), round(float64(b.Max.Z) / bs),
	}
}

// round strips the float32 noise (e.g. 0.30000001192 -> 0.3).
func round(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}
