package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf16"
)

// LatestProtocolVersion is the network protocol version this package targets
// (Luanti 5.18).
const LatestProtocolVersion = 54

// ErrShort is returned when a packet ends before all fields were read.
var ErrShort = errors.New("protocol: packet too short")

// Stream is a symmetric serializer: the same field-by-field code reads
// packets (a reading stream fills the pointed-to values) and writes them (a
// writing stream serializes the pointed-to values). Errors are sticky; check
// Err once at the end.
//
// All integers are big-endian, floats are IEEE-754 binary32, strings are a u16
// length followed by bytes ("long" strings use a u32 length) and wide strings
// are a u16 count of UTF-16 code units.
type Stream struct {
	// Reading reports whether this stream decodes (true) or encodes (false).
	Reading bool
	// ProtoVer is the negotiated protocol version; some layouts depend on it.
	ProtoVer uint16
	// Err is the first error encountered.
	Err error

	buf []byte
	pos int
}

// NewReader returns a stream decoding b.
func NewReader(b []byte, protoVer uint16) *Stream {
	return &Stream{Reading: true, ProtoVer: protoVer, buf: b}
}

// NewWriter returns a stream that encodes into an internal buffer.
func NewWriter(protoVer uint16) *Stream {
	return &Stream{ProtoVer: protoVer}
}

// Bytes returns what has been written so far.
func (s *Stream) Bytes() []byte { return s.buf }

// Remaining returns the number of unread bytes when reading.
func (s *Stream) Remaining() int {
	if !s.Reading {
		return 0
	}
	return len(s.buf) - s.pos
}

// More reports whether optional trailing fields are present. When writing it
// is always true, so optional fields are always emitted.
func (s *Stream) More() bool {
	if s.Err != nil {
		return false
	}
	return !s.Reading || s.pos < len(s.buf)
}

// Fail records an error unless one is already set.
func (s *Stream) Fail(err error) {
	if s.Err == nil {
		s.Err = err
	}
}

func (s *Stream) take(n int) []byte {
	if s.Err != nil {
		return nil
	}
	if n < 0 || s.pos+n > len(s.buf) {
		s.Err = ErrShort
		return nil
	}
	b := s.buf[s.pos : s.pos+n]
	s.pos += n
	return b
}

func (s *Stream) put(b ...byte) { s.buf = append(s.buf, b...) }

// U8 reads or writes a byte.
func (s *Stream) U8(p *uint8) {
	if s.Reading {
		if b := s.take(1); b != nil {
			*p = b[0]
		}
		return
	}
	s.put(*p)
}

// Bool reads or writes a boolean stored as one byte.
func (s *Stream) Bool(p *bool) {
	var v uint8
	if !s.Reading && *p {
		v = 1
	}
	s.U8(&v)
	if s.Reading {
		*p = v != 0
	}
}

// U16 reads or writes a big-endian u16.
func (s *Stream) U16(p *uint16) {
	if s.Reading {
		if b := s.take(2); b != nil {
			*p = binary.BigEndian.Uint16(b)
		}
		return
	}
	s.buf = binary.BigEndian.AppendUint16(s.buf, *p)
}

// S16 reads or writes a big-endian s16.
func (s *Stream) S16(p *int16) {
	v := uint16(*p)
	s.U16(&v)
	if s.Reading {
		*p = int16(v)
	}
}

// U32 reads or writes a big-endian u32.
func (s *Stream) U32(p *uint32) {
	if s.Reading {
		if b := s.take(4); b != nil {
			*p = binary.BigEndian.Uint32(b)
		}
		return
	}
	s.buf = binary.BigEndian.AppendUint32(s.buf, *p)
}

// S32 reads or writes a big-endian s32.
func (s *Stream) S32(p *int32) {
	v := uint32(*p)
	s.U32(&v)
	if s.Reading {
		*p = int32(v)
	}
}

// U64 reads or writes a big-endian u64.
func (s *Stream) U64(p *uint64) {
	if s.Reading {
		if b := s.take(8); b != nil {
			*p = binary.BigEndian.Uint64(b)
		}
		return
	}
	s.buf = binary.BigEndian.AppendUint64(s.buf, *p)
}

// F32 reads or writes an IEEE-754 binary32.
func (s *Stream) F32(p *float32) {
	v := math.Float32bits(*p)
	s.U32(&v)
	if s.Reading {
		*p = math.Float32frombits(v)
	}
}

// Str reads or writes a string with a u16 length prefix.
func (s *Stream) Str(p *string) {
	if s.Reading {
		var n uint16
		s.U16(&n)
		if b := s.take(int(n)); b != nil || n == 0 {
			*p = string(b)
		}
		return
	}
	if len(*p) > math.MaxUint16 {
		s.Fail(fmt.Errorf("protocol: string too long (%d)", len(*p)))
		return
	}
	n := uint16(len(*p))
	s.U16(&n)
	s.buf = append(s.buf, *p...)
}

// LStr reads or writes a string with a u32 length prefix.
func (s *Stream) LStr(p *string) {
	if s.Reading {
		var n uint32
		s.U32(&n)
		if b := s.take(int(n)); b != nil || n == 0 {
			*p = string(b)
		}
		return
	}
	n := uint32(len(*p))
	s.U32(&n)
	s.buf = append(s.buf, *p...)
}

// Blob reads or writes raw bytes with a u16 length prefix.
func (s *Stream) Blob(p *[]byte) {
	str := string(*p)
	s.Str(&str)
	if s.Reading {
		*p = []byte(str)
	}
}

// LBlob reads or writes raw bytes with a u32 length prefix.
func (s *Stream) LBlob(p *[]byte) {
	str := string(*p)
	s.LStr(&str)
	if s.Reading {
		*p = []byte(str)
	}
}

// WStr reads or writes a wide string (u16 count of UTF-16 units), exposed as
// a Go (UTF-8) string.
func (s *Stream) WStr(p *string) {
	if s.Reading {
		var n uint16
		s.U16(&n)
		b := s.take(int(n) * 2)
		if b == nil && n != 0 {
			return
		}
		u := make([]uint16, n)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(b[i*2:])
		}
		*p = string(utf16.Decode(u))
		return
	}
	u := utf16.Encode([]rune(*p))
	if len(u) > math.MaxUint16 {
		s.Fail(errors.New("protocol: wide string too long"))
		return
	}
	n := uint16(len(u))
	s.U16(&n)
	for _, c := range u {
		s.buf = binary.BigEndian.AppendUint16(s.buf, c)
	}
}

// Fixed reads or writes exactly len(p) raw bytes.
func (s *Stream) Fixed(p []byte) {
	if s.Reading {
		if b := s.take(len(p)); b != nil {
			copy(p, b)
		}
		return
	}
	s.buf = append(s.buf, p...)
}

// Rest reads all remaining bytes or writes the given bytes verbatim.
func (s *Stream) Rest(p *[]byte) {
	if s.Reading {
		if b := s.take(len(s.buf) - s.pos); b != nil || s.Err == nil {
			*p = append([]byte(nil), b...)
		}
		return
	}
	s.buf = append(s.buf, *p...)
}

// V2F reads or writes two floats.
func (s *Stream) V2F(p *V2F) { s.F32(&p.X); s.F32(&p.Y) }

// V3F reads or writes three floats.
func (s *Stream) V3F(p *V3F) { s.F32(&p.X); s.F32(&p.Y); s.F32(&p.Z) }

// V2S32 reads or writes two s32.
func (s *Stream) V2S32(p *V2S32) { s.S32(&p.X); s.S32(&p.Y) }

// V3S16 reads or writes three s16.
func (s *Stream) V3S16(p *V3S16) { s.S16(&p.X); s.S16(&p.Y); s.S16(&p.Z) }

// V3S32 reads or writes three s32.
func (s *Stream) V3S32(p *V3S32) { s.S32(&p.X); s.S32(&p.Y); s.S32(&p.Z) }

// Color reads or writes an ARGB color (u32).
func (s *Stream) Color(p *Color) { v := uint32(*p); s.U32(&v); *p = Color(v) }

// Strings16 reads or writes a list of strings preceded by a u16 count.
func (s *Stream) Strings16(p *[]string) {
	Slice16(s, p, func(e *string) { s.Str(e) })
}

func (s *Stream) readCount(n int) (int, bool) {
	// every element takes at least one byte, so a larger count is bogus
	if n > s.Remaining() {
		s.Fail(ErrShort)
		return 0, false
	}
	return n, true
}

// Slice8 reads or writes a slice preceded by a u8 count.
func Slice8[T any](s *Stream, p *[]T, elem func(*T)) {
	if s.Reading {
		var n uint8
		s.U8(&n)
		sliceRead(s, p, int(n), elem)
		return
	}
	if len(*p) > math.MaxUint8 {
		s.Fail(errors.New("protocol: too many elements for u8 count"))
		return
	}
	n := uint8(len(*p))
	s.U8(&n)
	for i := range *p {
		elem(&(*p)[i])
	}
}

// Slice16 reads or writes a slice preceded by a u16 count.
func Slice16[T any](s *Stream, p *[]T, elem func(*T)) {
	if s.Reading {
		var n uint16
		s.U16(&n)
		sliceRead(s, p, int(n), elem)
		return
	}
	if len(*p) > math.MaxUint16 {
		s.Fail(errors.New("protocol: too many elements for u16 count"))
		return
	}
	n := uint16(len(*p))
	s.U16(&n)
	for i := range *p {
		elem(&(*p)[i])
	}
}

// Slice32 reads or writes a slice preceded by a u32 count.
func Slice32[T any](s *Stream, p *[]T, elem func(*T)) {
	if s.Reading {
		var n uint32
		s.U32(&n)
		sliceRead(s, p, int(n), elem)
		return
	}
	n := uint32(len(*p))
	s.U32(&n)
	for i := range *p {
		elem(&(*p)[i])
	}
}

// SliceRest reads elements until the stream is exhausted (or writes them all).
func SliceRest[T any](s *Stream, p *[]T, elem func(*T)) {
	if s.Reading {
		*p = nil
		for s.Err == nil && s.Remaining() > 0 {
			var e T
			elem(&e)
			if s.Err == nil {
				*p = append(*p, e)
			}
		}
		return
	}
	for i := range *p {
		elem(&(*p)[i])
	}
}

func sliceRead[T any](s *Stream, p *[]T, n int, elem func(*T)) {
	n, ok := s.readCount(n)
	if !ok {
		return
	}
	out := make([]T, n)
	for i := range out {
		elem(&out[i])
		if s.Err != nil {
			return
		}
	}
	*p = out
}

// PackedStrings reads or writes Luanti's "string16 array": a u32 count, then
// all u16 lengths, then all string contents back to back.
func (s *Stream) PackedStrings(p *[]string) {
	if !s.Reading {
		n := uint32(len(*p))
		s.U32(&n)
		for _, e := range *p {
			if len(e) > 0xffff {
				s.Fail(errors.New("protocol: string too long"))
				return
			}
			l := uint16(len(e))
			s.U16(&l)
		}
		for _, e := range *p {
			s.buf = append(s.buf, e...)
		}
		return
	}
	var n uint32
	s.U32(&n)
	cnt, ok := s.readCount(int(n))
	if !ok {
		return
	}
	lens := make([]uint16, cnt)
	for i := range lens {
		s.U16(&lens[i])
	}
	out := make([]string, cnt)
	for i, l := range lens {
		if b := s.take(int(l)); b != nil || l == 0 {
			out[i] = string(b)
		}
	}
	if s.Err == nil {
		*p = out
	}
}
