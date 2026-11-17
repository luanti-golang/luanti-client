package mapdump

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/luanti-golang/mapparser"

	"luanti-client/protocol"
)

// testdata/blockdata-v29.bin is ToClientBlockData.Data captured from a Luanti
// 5.18 server (serialization version 29).
func loadBlock(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/blockdata-v29.bin")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// decode reverses Convert's payload encoding.
func decode(t *testing.T, mb *Mapblock) (ids []uint16, p1, p2 []byte) {
	t.Helper()
	compressed, err := base64.RawStdEncoding.DecodeString(mb.Mapdata)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != nodeCount*4 {
		t.Fatalf("payload length %d", len(raw))
	}
	ids = make([]uint16, nodeCount)
	for i := range ids {
		ids[i] = binary.LittleEndian.Uint16(raw[i*2:])
	}
	return ids, raw[nodeCount*2 : nodeCount*3], raw[nodeCount*3:]
}

func TestConvert(t *testing.T) {
	data := loadBlock(t)
	want, err := mapparser.ParseNetwork(29, data)
	if err != nil {
		t.Fatal(err)
	}

	// name every id but one, which must come out as UnknownName
	names := map[uint16]string{}
	var unknown uint16
	for i, cid := range want.ContentId {
		if i == 0 {
			unknown = uint16(cid)
			continue
		}
		if uint16(cid) != unknown {
			names[uint16(cid)] = fmt.Sprintf("test:node%d", cid)
		}
	}

	mb, err := Convert(29, data, names)
	if err != nil {
		t.Fatal(err)
	}
	ids, p1, p2 := decode(t, mb)
	for i := 0; i < nodeCount; i++ {
		if int(ids[i]) != want.ContentId[i] || int(p1[i]) != want.Param1[i] || int(p2[i]) != want.Param2[i] {
			t.Fatalf("node %d: got %d/%d/%d, want %d/%d/%d", i, ids[i], p1[i], p2[i],
				want.ContentId[i], want.Param1[i], want.Param2[i])
		}
	}

	// node_mapping holds exactly the used ids, every one with a unique name
	used := map[uint16]bool{}
	for _, id := range ids {
		used[id] = true
	}
	if len(mb.NodeMapping) != len(used) {
		t.Errorf("node_mapping has %d entries, block uses %d ids", len(mb.NodeMapping), len(used))
	}
	for name, id := range mb.NodeMapping {
		if !used[id] {
			t.Errorf("%s -> %d is not used in the block", name, id)
		}
		if id == unknown && name != UnknownName(unknown) {
			t.Errorf("unknown id %d named %q", id, name)
		}
		if id != unknown && name != names[id] {
			t.Errorf("id %d named %q, want %q", id, name, names[id])
		}
	}
}

func TestConvertError(t *testing.T) {
	if _, err := Convert(29, []byte{1, 2, 3}, nil); err == nil {
		t.Error("expected an error for garbage data")
	}
}

func TestNames(t *testing.T) {
	names := Names([]protocol.NodeDefinition{{ID: 10, Name: "default:stone"}})
	want := map[uint16]string{10: "default:stone", 125: "unknown", 126: "air", 127: "ignore"}
	if len(names) != len(want) {
		t.Fatalf("got %v", names)
	}
	for id, name := range want {
		if names[id] != name {
			t.Errorf("%d: got %q, want %q", id, names[id], name)
		}
	}
}

func TestFileName(t *testing.T) {
	if got := FileName(protocol.V3S16{X: -3, Y: 0, Z: 12}); got != "(-3,0,12).json" {
		t.Errorf("got %q", got)
	}
}

func TestDumper(t *testing.T) {
	data := loadBlock(t)
	parsed, err := mapparser.ParseNetwork(29, data)
	if err != nil {
		t.Fatal(err)
	}
	defs := func(prefix string) []protocol.NodeDefinition {
		var out []protocol.NodeDefinition
		seen := map[int]bool{}
		for _, cid := range parsed.ContentId {
			if !seen[cid] {
				seen[cid] = true
				out = append(out, protocol.NodeDefinition{ID: uint16(cid), Name: fmt.Sprintf("%s:%d", prefix, cid)})
			}
		}
		return out
	}

	dir := filepath.Join(t.TempDir(), "mapblocks")
	d, err := New(dir, t.Logf)
	if err != nil {
		t.Fatal(err)
	}

	// blocks before the node definitions are held back, a resent block
	// replaces the file, garbage is reported and skipped
	pos := protocol.V3S16{X: 1, Y: -2, Z: 3}
	d.Block(29, pos, data)
	d.Block(29, protocol.V3S16{}, []byte("garbage"))
	d.SetNodeDefs(defs("first"))
	d.SetNodeDefs(defs("second"))
	d.Block(29, pos, data)
	d.Close()
	d.Close() // idempotent
	d.Block(29, protocol.V3S16{X: 9}, data)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "(1,-2,3).json" {
		t.Fatalf("unexpected files: %v", entries)
	}
	b, err := os.ReadFile(filepath.Join(dir, "(1,-2,3).json"))
	if err != nil {
		t.Fatal(err)
	}
	var mb Mapblock
	if err := json.Unmarshal(b, &mb); err != nil {
		t.Fatal(err)
	}
	for name, id := range mb.NodeMapping {
		if _, builtin := builtinNames[id]; !builtin && name != fmt.Sprintf("second:%d", id) {
			t.Errorf("%q -> %d, want the names of the last SetNodeDefs", name, id)
		}
	}
}

func TestDumperWithoutNodeDefs(t *testing.T) {
	dir := t.TempDir()
	d, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	d.Block(29, protocol.V3S16{}, loadBlock(t))
	d.Close()

	b, err := os.ReadFile(filepath.Join(dir, "(0,0,0).json"))
	if err != nil {
		t.Fatal(err)
	}
	var mb Mapblock
	if err := json.Unmarshal(b, &mb); err != nil {
		t.Fatal(err)
	}
	for name, id := range mb.NodeMapping {
		if name != UnknownName(id) {
			t.Errorf("%q -> %d, want %q", name, id, UnknownName(id))
		}
	}
}
