package transport

import (
	"encoding/binary"
	"fmt"
)

// Describe renders the MTP framing of a raw datagram as a short human-readable
// string, e.g. "peer=2 ch=1 RELIABLE seq=65501 ORIGINAL 31B".
func Describe(d []byte) string {
	if len(d) < baseHeader {
		return fmt.Sprintf("short datagram (%d bytes)", len(d))
	}
	if binary.BigEndian.Uint32(d) != protocolID {
		return fmt.Sprintf("bad protocol id %#x", binary.BigEndian.Uint32(d))
	}
	s := fmt.Sprintf("peer=%d ch=%d", binary.BigEndian.Uint16(d[4:]), d[6])
	return s + " " + describeInner(d[baseHeader:])
}

func describeInner(p []byte) string {
	if len(p) == 0 {
		return "empty"
	}
	switch p[0] {
	case TypeControl:
		if len(p) < 2 {
			return "CONTROL (truncated)"
		}
		switch p[1] {
		case ControlAck:
			if len(p) >= 4 {
				return fmt.Sprintf("CONTROL ACK seq=%d", binary.BigEndian.Uint16(p[2:]))
			}
		case ControlSetPeer:
			if len(p) >= 4 {
				return fmt.Sprintf("CONTROL SET_PEER_ID id=%d", binary.BigEndian.Uint16(p[2:]))
			}
		case ControlPing:
			return "CONTROL PING"
		case ControlDisco:
			return "CONTROL DISCO"
		}
		return fmt.Sprintf("CONTROL type=%d", p[1])
	case TypeOriginal:
		return fmt.Sprintf("ORIGINAL %dB", len(p)-1)
	case TypeSplit:
		if len(p) < 7 {
			return "SPLIT (truncated)"
		}
		return fmt.Sprintf("SPLIT seq=%d chunk=%d/%d %dB", binary.BigEndian.Uint16(p[1:]),
			binary.BigEndian.Uint16(p[5:])+1, binary.BigEndian.Uint16(p[3:]), len(p)-7)
	case TypeReliable:
		if len(p) < 3 {
			return "RELIABLE (truncated)"
		}
		return fmt.Sprintf("RELIABLE seq=%d > %s", binary.BigEndian.Uint16(p[1:]), describeInner(p[3:]))
	}
	return fmt.Sprintf("unknown type %d", p[0])
}
