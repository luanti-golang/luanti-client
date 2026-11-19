// Package mapdump saves map blocks received from a server (TOCLIENT_BLOCKDATA)
// as files in the mapblock format of luanti-viewer (parseMapblock in its
// src/mapblock.ts), one file per block named "(x,y,z).json":
//
//	{"mapdata": "<base64>", "node_mapping": {"default:stone": 12, ...}}
//
// mapdata is the unpadded base64 of a zlib stream of 16384 bytes: 4096 content
// ids (uint16, little endian), 4096 param1 and 4096 param2 bytes, in VoxelArea
// order (index = z*256 + y*16 + x). node_mapping contains the nodes used in the
// block.
//
// Blocks are parsed with github.com/luanti-golang/mapparser. Over the network
// they carry the server's global content ids instead of a per-block name
// mapping, so the names come from the node definitions (TOCLIENT_NODEDEF).
// Node metadata (inventories, fields) is not part of the format and dropped.
package mapdump

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/luanti-golang/mapparser"

	"github.com/luanti-golang/luanti-client/protocol"
)

// nodeCount is the number of nodes in a map block (16^3).
const nodeCount = 4096

// Mapblock is one exported map block.
type Mapblock struct {
	Mapdata     string            `json:"mapdata"`
	NodeMapping map[string]uint16 `json:"node_mapping"`
}

// FileName is the name of the file a block at pos (in map block coordinates)
// is saved as.
func FileName(pos protocol.V3S16) string {
	return fmt.Sprintf("(%d,%d,%d).json", pos.X, pos.Y, pos.Z)
}

// builtinNames are Luanti's built-in content ids (CONTENT_UNKNOWN, CONTENT_AIR,
// CONTENT_IGNORE), which servers do not send node definitions for.
var builtinNames = map[uint16]string{125: "unknown", 126: "air", 127: "ignore"}

// Names returns the content id -> node name table of the node definitions,
// including the built-in nodes "unknown", "air" and "ignore".
func Names(defs []protocol.NodeDefinition) map[uint16]string {
	names := make(map[uint16]string, len(defs)+len(builtinNames))
	for id, name := range builtinNames {
		names[id] = name
	}
	for _, d := range defs {
		names[d.ID] = d.Name
	}
	return names
}

// UnknownName is the node name used for content ids without a definition.
// Names must be unique in node_mapping, so the id is part of it.
func UnknownName(id uint16) string {
	return fmt.Sprintf("unknown:%d", id)
}

// Convert parses a TOCLIENT_BLOCKDATA payload (ToClientBlockData.Data) that
// was serialized with the given serialization version (from TOCLIENT_HELLO)
// and converts it. names maps content ids to node names, see Names.
func Convert(serVer uint8, data []byte, names map[uint16]string) (*Mapblock, error) {
	mb, err := mapparser.ParseNetwork(serVer, data)
	if err != nil {
		return nil, err
	}
	if len(mb.ContentId) != nodeCount || len(mb.Param1) != nodeCount || len(mb.Param2) != nodeCount {
		return nil, fmt.Errorf("unexpected node count %d", len(mb.ContentId))
	}

	raw := make([]byte, nodeCount*4)
	mapping := map[string]uint16{}
	for i, cid := range mb.ContentId {
		id := uint16(cid)
		binary.LittleEndian.PutUint16(raw[i*2:], id)
		raw[nodeCount*2+i] = byte(mb.Param1[i])
		raw[nodeCount*3+i] = byte(mb.Param2[i])
		name, ok := names[id]
		if !ok {
			name = UnknownName(id)
		}
		mapping[name] = id
	}

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &Mapblock{
		Mapdata:     base64.RawStdEncoding.EncodeToString(buf.Bytes()),
		NodeMapping: mapping,
	}, nil
}

type job struct {
	serVer uint8
	pos    protocol.V3S16
	data   []byte
	names  map[uint16]string
}

// Dumper saves blocks into a directory in the background. Blocks that arrive
// before the node definitions are held back until SetNodeDefs is called. A
// block that is sent again (because it changed) replaces the file.
type Dumper struct {
	dir  string
	logf func(format string, args ...any)

	mu      sync.Mutex // guards the fields below
	names   map[uint16]string
	pending []job
	closed  bool

	queue chan job
	done  chan struct{}
}

// New creates dir if needed and starts a Dumper writing into it. logf
// receives errors (nil: discarded). Call Close to write the remaining blocks.
func New(dir string, logf func(format string, args ...any)) (*Dumper, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	d := &Dumper{
		dir:   dir,
		logf:  logf,
		queue: make(chan job, 256),
		done:  make(chan struct{}),
	}
	go d.run()
	return d, nil
}

// SetNodeDefs sets the node definitions used to name content ids and releases
// the blocks held back so far.
func (d *Dumper) SetNodeDefs(defs []protocol.NodeDefinition) {
	names := Names(defs)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.names = names
	for _, j := range d.pending {
		j.names = names
		d.queue <- j
	}
	d.pending = nil
}

// Block queues a block for saving. data is copied.
func (d *Dumper) Block(serVer uint8, pos protocol.V3S16, data []byte) {
	j := job{serVer: serVer, pos: pos, data: bytes.Clone(data)}
	// The lock is held while queueing so that blocks are written in the order
	// they arrived; the writer never takes it, so this cannot deadlock.
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case d.closed:
	case d.names == nil:
		d.pending = append(d.pending, j)
	default:
		j.names = d.names
		d.queue <- j
	}
}

// Close writes all queued blocks and stops the Dumper. Blocks still waiting
// for node definitions are written with UnknownName for every content id.
func (d *Dumper) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		<-d.done
		return
	}
	d.closed = true
	if len(d.pending) > 0 {
		d.logf("mapblocks: no node definitions received, writing %d blocks without node names", len(d.pending))
		for _, j := range d.pending {
			j.names = map[uint16]string{}
			d.queue <- j
		}
		d.pending = nil
	}
	close(d.queue)
	d.mu.Unlock()
	<-d.done
}

func (d *Dumper) run() {
	defer close(d.done)
	for j := range d.queue {
		if err := d.write(j); err != nil {
			d.logf("mapblock %s: %v", FileName(j.pos), err)
		}
	}
}

func (d *Dumper) write(j job) error {
	mb, err := Convert(j.serVer, j.data, j.names)
	if err != nil {
		return err
	}
	b, err := json.Marshal(mb)
	if err != nil {
		return err
	}
	path := filepath.Join(d.dir, FileName(j.pos))
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
