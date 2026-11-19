package viewerdef

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/luanti-golang/luanti-client/protocol"
)

func tile(name string) protocol.TileDef {
	return protocol.TileDef{Name: name, BackfaceCulling: true, AlignStyle: "node"}
}

func aabb(x1, y1, z1, x2, y2, z2 float32) protocol.AABB {
	return protocol.AABB{Min: protocol.V3F{X: x1, Y: y1, Z: z1}, Max: protocol.V3F{X: x2, Y: y2, Z: z2}}
}

func base(id uint16, name, drawtype string) protocol.NodeDefinition {
	return protocol.NodeDefinition{
		ID: id, Name: name, DrawType: drawtype, ParamType: "none", ParamType2: "none",
		Alpha: "opaque", VisualScale: 1, Color: protocol.RGB{R: 255, G: 255, B: 255}, LiquidRange: 8,
		NodeBox: protocol.NodeBox{Type: "regular"},
	}
}

func TestConvert(t *testing.T) {
	stone := base(10, "default:stone", "normal")
	stone.Groups = map[string]int16{"cracky": 3}
	stone.Walkable = true
	for i := range stone.Tiles {
		stone.Tiles[i] = tile("default_stone.png")
	}

	fence := base(11, "default:fence_wood", "nodebox")
	fence.ParamType = "light"
	fence.ParamType2 = "full"
	fence.Tiles[0] = tile("default_fence_wood.png")
	fence.ConnectsToID = []uint16{10, 11, 999}
	fence.NodeBox = protocol.NodeBox{
		Type:  "connected",
		Fixed: []protocol.AABB{aabb(-1.25, -5, -1.25, 1.25, 5, 1.25)},
		Connected: &protocol.ConnectedBox{
			ConnectFront: []protocol.AABB{aabb(-0.625, 1.875, -5, 0.625, 3.125, -1.25)},
		},
	}

	water := base(12, "default:water_flowing", "flowingliquid")
	water.Alpha = "blend"
	water.Color = protocol.RGB{R: 0x10, G: 0x20, B: 0x30}
	water.PostEffect = 0x671e3c5a
	water.LiquidAlternativeSource = "default:water_source"
	water.Tiles[0] = tile("default_water.png")
	anim := &protocol.TileAnimation{Type: "vertical_frames", AspectW: 16, AspectH: 16, Length: 0.5}
	water.SpecialTiles[0] = protocol.TileDef{Name: "water_anim.png", Animation: anim, AlignStyle: "world"}
	water.SpecialTiles[1] = protocol.TileDef{Name: "water_anim.png", BackfaceCulling: true, Animation: anim}

	vent := base(13, "test:vent", "airlike")
	vent.Mesh = "ignored.obj"

	got := Convert([]protocol.NodeDefinition{stone, fence, water, vent})
	if len(got) != 5 {
		t.Fatalf("got %d definitions, want 5 (with air)", len(got))
	}
	if got["air"].DrawType != "airlike" || got["air"].Walkable {
		t.Errorf("air: %+v", got["air"])
	}

	s := got["default:stone"]
	if len(s.Tiles) != 6 || s.Tiles[5].Name != "default_stone.png" || !s.Tiles[0].BackfaceCulling ||
		s.Tiles[0].AlignStyle != "" || s.Color != "" || s.Groups["cracky"] != 3 || !s.Walkable ||
		s.NodeBox != nil || s.UseTextureAlpha != "opaque" {
		t.Errorf("stone: %+v", s)
	}

	f := got["default:fence_wood"]
	if f.ParamType != "light" || f.ParamType2 != "none" || f.Groups != nil {
		t.Errorf("fence params: %+v", f)
	}
	if !reflect.DeepEqual(f.ConnectsTo, []string{"default:stone", "default:fence_wood"}) {
		t.Errorf("connects_to = %v", f.ConnectsTo)
	}
	if len(f.Tiles) != 1 {
		t.Errorf("fence tiles: trailing untextured tiles not trimmed: %d", len(f.Tiles))
	}
	wantBox := &NodeBox{
		Type:         "connected",
		Fixed:        []Box{{-0.125, -0.5, -0.125, 0.125, 0.5, 0.125}},
		ConnectFront: []Box{{-0.0625, 0.1875, -0.5, 0.0625, 0.3125, -0.125}},
	}
	if !reflect.DeepEqual(f.NodeBox, wantBox) {
		t.Errorf("node_box = %+v, want %+v", f.NodeBox, wantBox)
	}

	w := got["default:water_flowing"]
	if w.Color != "#102030" || w.UseTextureAlpha != "blend" || w.LiquidAlternativeSource != "default:water_source" {
		t.Errorf("water: %+v", w)
	}
	if w.PostEffectColor == nil || *w.PostEffectColor != (Color{R: 0x1e, G: 0x3c, B: 0x5a, A: 0x67}) {
		t.Errorf("post_effect_color = %+v", w.PostEffectColor)
	}
	if len(w.SpecialTiles) != 2 || w.SpecialTiles[0].BackfaceCulling || !w.SpecialTiles[1].BackfaceCulling ||
		w.SpecialTiles[0].AlignStyle != "world" || w.SpecialTiles[0].Animation == nil {
		t.Errorf("special_tiles = %+v", w.SpecialTiles)
	}

	v := got["test:vent"]
	if v.Tiles != nil || v.Mesh != "" {
		t.Errorf("airlike: %+v", v)
	}
}

func TestJSONShape(t *testing.T) {
	d := base(1, "test:slab", "nodebox")
	d.Tiles[0] = tile("a.png")
	d.Tiles[0].Animation = &protocol.TileAnimation{Type: "sheet_2d", FramesW: 2, FramesH: 4, FrameLength: 0.25}
	d.NodeBox = protocol.NodeBox{Type: "fixed", Fixed: []protocol.AABB{aabb(-5, -5, -5, 5, 0, 5)}}
	b, err := json.Marshal(ConvertOne(&d, nil))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	// required fields of the viewer type are present, groups is null
	for _, k := range []string{"name", "drawtype", "paramtype", "paramtype2", "light_source", "groups"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %q in %s", k, b)
		}
	}
	if m["groups"] != nil {
		t.Errorf("groups = %v, want null", m["groups"])
	}
	want := `{"fixed":[[-0.5,-0.5,-0.5,0.5,0,0.5]],"type":"fixed"}`
	if got, _ := json.Marshal(m["node_box"]); string(got) != want {
		t.Errorf("node_box = %s, want %s", got, want)
	}
	wantAnim := `{"frame_length":0.25,"frames_h":4,"frames_w":2,"type":"sheet_2d"}`
	if got, _ := json.Marshal(m["tiles"].([]any)[0].(map[string]any)["animation"]); string(got) != wantAnim {
		t.Errorf("animation = %s, want %s", got, wantAnim)
	}
}

func TestBoxRounding(t *testing.T) {
	// 0.3 node = 3 BS, which is not exact in float32
	got := box(aabb(-3, 0.1*10, 0, 3, 4.5, 1.0/16*10))
	want := Box{-0.3, 0.1, 0, 0.3, 0.45, 0.0625}
	if got != want {
		t.Errorf("box = %v, want %v", got, want)
	}
}
