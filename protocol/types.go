package protocol

import "fmt"

// V2F is a 2D float vector.
type V2F struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// V3F is a 3D float vector. Positions sent by the server are in "BS" units:
// one node is 10 units.
type V3F struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// V2S32 is a 2D integer vector.
type V2S32 struct{ X, Y int32 }

// V3S16 is a 3D integer vector, used for node and map block positions.
type V3S16 struct{ X, Y, Z int16 }

// V3S32 is a 3D integer vector, used for fixed-point player positions.
type V3S32 struct{ X, Y, Z int32 }

// Color is an ARGB color packed into a u32 (0xAARRGGBB).
type Color uint32

func (c Color) String() string { return fmt.Sprintf("#%08x", uint32(c)) }

// MarshalText makes colors appear as "#aarrggbb" in JSON.
func (c Color) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// MapNode is a serialized map node (serialization format 29).
type MapNode struct {
	Param0 uint16 // content id
	Param1 uint8  // light
	Param2 uint8
}

func (n *MapNode) sync(s *Stream) { s.U16(&n.Param0); s.U8(&n.Param1); s.U8(&n.Param2) }
