package protocol

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

func TestHelloKnownBytes(t *testing.T) {
	// serialization 29, compression 0, proto 54, auth SRP|FIRST_SRP.. name ""
	raw := []byte{0x00, 0x02, 29, 0, 0, 0, 54, 0, 0, 0, 2, 0, 0}
	p, err := DecodeToClient(raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := p.(*ToClientHello)
	if h.SerializationVersion != 29 || h.ProtocolVersion != 54 || h.AuthMethods != AuthSRP {
		t.Fatalf("bad decode: %+v", h)
	}
	out, err := Encode(h, 0)
	if err != nil || !bytes.Equal(out, raw) {
		t.Fatalf("re-encode mismatch: %x err=%v", out, err)
	}
}

func TestChatMessageUTF16(t *testing.T) {
	m := &ToClientChatMessage{Version: 1, Type: ChatNormal, Sender: "", Message: "<héllo> 😀", Timestamp: 5}
	b, err := Encode(m, 0)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodeToClient(b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.(*ToClientChatMessage); got.Message != m.Message {
		t.Fatalf("got %q", got.Message)
	}
}

func TestPlayerPosLayout(t *testing.T) {
	p := &ToServerPlayerPos{}
	p.Position = V3S32{100, 200, 300}
	b, _ := Encode(p, 0)
	// cmd(2) + pos(12) + speed(12) + pitch(4) + yaw(4) + keys(4) + fov(1) + range(1) + inverted(1) + 2 floats(8)
	if len(b) != 2+12+12+4+4+4+1+1+1+8 {
		t.Fatalf("length %d", len(b))
	}
}

// Every registered packet, filled with plausible data, must survive an
// encode/decode round trip.
func TestRoundTripAll(t *testing.T) {
	for _, dir := range []Direction{ToClient, ToServer} {
		reg := toClientReg
		if dir == ToServer {
			reg = toServerReg
		}
		for cmd, mk := range reg {
			p := mk()
			fill(reflect.ValueOf(p).Elem(), rand.New(rand.NewSource(int64(cmd))), p)
			fixDiscriminators(p)
			for _, ver := range []uint16{47, 54} {
				b, err := Encode(p, ver)
				if err != nil {
					t.Errorf("%s: encode: %v", Name(p), err)
					continue
				}
				var q Packet
				if dir == ToClient {
					q, err = DecodeToClient(b, ver)
				} else {
					q, err = DecodeToServer(b, ver)
				}
				if err != nil {
					t.Errorf("%s v%d: decode: %v", Name(p), ver, err)
					continue
				}
				if ver == 54 && !reflect.DeepEqual(p, q) {
					t.Errorf("%s: mismatch\n  in : %+v\n  out: %+v", Name(p), p, q)
				}
				b2, _ := Encode(q, ver)
				if !bytes.Equal(b, b2) {
					t.Errorf("%s v%d: re-encode differs", Name(p), ver)
				}
			}
		}
	}
}

// Decoding garbage must never panic.
func TestDecodeGarbage(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		n := r.Intn(64)
		b := make([]byte, n)
		r.Read(b)
		if n >= 2 {
			// bias towards valid commands
			b[0] = 0
			b[1] = byte(r.Intn(0x66))
		}
		_, _ = DecodeToClient(b, 0)
		_, _ = DecodeToServer(b, 0)
	}
}

func fill(v reflect.Value, r *rand.Rand, p Packet) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fill(v.Field(i), r, p)
			}
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(r.Intn(100)))
	case reflect.Int16, reflect.Int32:
		v.SetInt(int64(r.Intn(100)))
	case reflect.Float32:
		v.SetFloat(float64(r.Intn(1000)) / 4)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString("héllo")
	case reflect.Slice:
		n := 2
		v.Set(reflect.MakeSlice(v.Type(), n, n))
		for i := 0; i < n; i++ {
			fill(v.Index(i), r, p)
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fill(v.Index(i), r, p)
		}
	}
}

// fixDiscriminators sets selector fields to valid values so that the fields
// they guard are actually serialized.
func fixDiscriminators(p Packet) {
	switch q := p.(type) {
	case *ToClientHudChange:
		q.Stat = HudStatPos
		q.String, q.V3F, q.Int = "", V3F{}, 0
	case *ToClientSetSky:
		q.Type = "regular"
		q.Textures = nil
	case *ToClientModChannelSignal:
		q.Signal = ModChannelSetState
	case *ToServerInteract:
		q.Target.Type = PointedNode
		q.Target.ObjectID = 0
	}
}
