// Package protocol defines every packet of the Luanti network protocol
// (protocol version 54, Luanti 5.18) as a Go struct, together with
// encoding and decoding.
//
// A packet on the wire is a big-endian u16 command followed by the command's
// data. Use DecodeToClient / DecodeToServer to turn bytes into a Packet and
// Encode to turn a Packet back into bytes.
package protocol

import (
	"fmt"
	"sort"
)

// Direction is the direction a packet travels.
type Direction int

const (
	// ToClient packets are sent by the server.
	ToClient Direction = iota
	// ToServer packets are sent by the client.
	ToServer
)

func (d Direction) String() string {
	if d == ToServer {
		return "C->S"
	}
	return "S->C"
}

// Packet is implemented by every packet struct.
type Packet interface {
	// Command returns the wire command number.
	Command() uint16
	// Direction says whether the server or the client sends this packet.
	Direction() Direction
	// Sync reads or writes the packet's data (without the command).
	Sync(s *Stream)
}

// Unknown holds a packet whose command is not known, or that failed to decode.
type Unknown struct {
	Dir  Direction
	Cmd  uint16
	Data []byte
}

func (p *Unknown) Command() uint16      { return p.Cmd }
func (p *Unknown) Direction() Direction { return p.Dir }
func (p *Unknown) Sync(s *Stream)       { s.Rest(&p.Data) }

type registry map[uint16]func() Packet

var (
	toClientReg = registry{}
	toServerReg = registry{}
	nameOf      = map[Direction]map[uint16]string{ToClient: {}, ToServer: {}}
)

func register[T any, PT interface {
	*T
	Packet
}](dir Direction, name string) {
	cmd := PT(new(T)).Command()
	f := func() Packet { return PT(new(T)) }
	if dir == ToClient {
		toClientReg[cmd] = f
	} else {
		toServerReg[cmd] = f
	}
	nameOf[dir][cmd] = name
}

// Name returns the Luanti constant name of a packet, e.g. "TOCLIENT_HELLO".
func Name(p Packet) string { return CommandName(p.Direction(), p.Command()) }

// CommandName returns the Luanti constant name for a command number.
func CommandName(dir Direction, cmd uint16) string {
	if n, ok := nameOf[dir][cmd]; ok {
		return n
	}
	if dir == ToClient {
		return fmt.Sprintf("TOCLIENT_UNKNOWN_%#02x", cmd)
	}
	return fmt.Sprintf("TOSERVER_UNKNOWN_%#02x", cmd)
}

// CommandNames lists the known command names of a direction, sorted by number.
func CommandNames(dir Direction) []string {
	var cmds []int
	for c := range nameOf[dir] {
		cmds = append(cmds, int(c))
	}
	sort.Ints(cmds)
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = nameOf[dir][uint16(c)]
	}
	return out
}

// Encode serializes a packet to wire bytes (u16 command + data) using the
// given protocol version (0 means latest).
func Encode(p Packet, protoVer uint16) ([]byte, error) {
	if protoVer == 0 {
		protoVer = LatestProtocolVersion
	}
	s := NewWriter(protoVer)
	cmd := p.Command()
	s.U16(&cmd)
	p.Sync(s)
	if s.Err != nil {
		return nil, fmt.Errorf("encode %s: %w", Name(p), s.Err)
	}
	return s.Bytes(), nil
}

// DecodeToClient decodes a packet received from the server. protoVer is the
// negotiated protocol version (0 means latest). Packets that are unknown or
// malformed are returned as *Unknown (together with an error for the latter).
func DecodeToClient(payload []byte, protoVer uint16) (Packet, error) {
	return decode(ToClient, toClientReg, payload, protoVer)
}

// DecodeToServer decodes a packet that a client sent to the server.
func DecodeToServer(payload []byte, protoVer uint16) (Packet, error) {
	return decode(ToServer, toServerReg, payload, protoVer)
}

func decode(dir Direction, reg registry, payload []byte, protoVer uint16) (Packet, error) {
	if len(payload) < 2 {
		return &Unknown{Dir: dir, Data: payload}, ErrShort
	}
	if protoVer == 0 {
		protoVer = LatestProtocolVersion
	}
	cmd := uint16(payload[0])<<8 | uint16(payload[1])
	mk, ok := reg[cmd]
	if !ok {
		return &Unknown{Dir: dir, Cmd: cmd, Data: append([]byte(nil), payload[2:]...)}, nil
	}
	p := mk()
	s := NewReader(payload[2:], protoVer)
	p.Sync(s)
	if s.Err != nil {
		return &Unknown{Dir: dir, Cmd: cmd, Data: append([]byte(nil), payload[2:]...)},
			fmt.Errorf("decode %s: %w", CommandName(dir, cmd), s.Err)
	}
	return p, nil
}
