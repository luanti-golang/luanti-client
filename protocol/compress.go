package protocol

import (
	"bytes"
	"compress/zlib"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

const maxDecompressed = 256 << 20

var (
	zstdOnce sync.Once
	zstdDec  *zstd.Decoder
	zstdEnc  *zstd.Encoder
)

func initZstd() {
	zstdOnce.Do(func() {
		zstdDec, _ = zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxDecompressed))
		zstdEnc, _ = zstd.NewWriter(nil)
	})
}

// DecompressZstd decompresses a zstd frame.
func DecompressZstd(b []byte) ([]byte, error) {
	initZstd()
	return zstdDec.DecodeAll(b, nil)
}

// CompressZstd compresses data into a zstd frame.
func CompressZstd(b []byte) []byte {
	initZstd()
	return zstdEnc.EncodeAll(b, nil)
}

// DecompressZlib decompresses a zlib stream.
func DecompressZlib(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, maxDecompressed))
}

// CompressZlib compresses data into a zlib stream.
func CompressZlib(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// definitionCodec decompresses (reading) or compresses (writing) a node/item
// definition blob: zstd since protocol 48, zlib before.
func (s *Stream) compressedBlob(p *[]byte) {
	if s.Reading {
		var raw []byte
		s.LBlob(&raw)
		if s.Err != nil {
			return
		}
		var out []byte
		var err error
		if s.ProtoVer >= 48 {
			out, err = DecompressZstd(raw)
		} else {
			out, err = DecompressZlib(raw)
		}
		if err != nil {
			s.Fail(err)
			return
		}
		*p = out
		return
	}
	var c []byte
	if s.ProtoVer >= 48 {
		c = CompressZstd(*p)
	} else {
		c = CompressZlib(*p)
	}
	s.LBlob(&c)
}
