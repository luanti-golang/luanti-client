package transport

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// fakeServer speaks just enough MTP to exercise the client.
type fakeServer struct {
	t    *testing.T
	udp  *net.UDPConn
	peer *net.UDPAddr
	id   uint16
}

func newFakeServer(t *testing.T) *fakeServer {
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { udp.Close() })
	return &fakeServer{t: t, udp: udp, id: 77}
}

func (s *fakeServer) addr() string { return s.udp.LocalAddr().String() }

func (s *fakeServer) read(timeout time.Duration) []byte {
	_ = s.udp.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 2048)
	n, from, err := s.udp.ReadFromUDP(buf)
	if err != nil {
		return nil
	}
	s.peer = from
	return buf[:n]
}

func (s *fakeServer) send(channel uint8, inner []byte) {
	b := make([]byte, 7+len(inner))
	binary.BigEndian.PutUint32(b, protocolID)
	binary.BigEndian.PutUint16(b[4:], PeerIDServer)
	b[6] = channel
	copy(b[7:], inner)
	if _, err := s.udp.WriteToUDP(b, s.peer); err != nil {
		s.t.Fatal(err)
	}
}

func reliable(seq uint16, inner []byte) []byte {
	return append([]byte{TypeReliable, byte(seq >> 8), byte(seq)}, inner...)
}

func original(payload []byte) []byte { return append([]byte{TypeOriginal}, payload...) }

func split(seq uint16, count, num int, data []byte) []byte {
	h := []byte{TypeSplit, byte(seq >> 8), byte(seq), byte(count >> 8), byte(count), byte(num >> 8), byte(num)}
	return append(h, data...)
}

// handshake answers the client's connect packet and returns the connection.
func (s *fakeServer) handshake() *Conn {
	type res struct {
		c   *Conn
		err error
	}
	ch := make(chan res, 1)
	go func() {
		c, err := Dial(s.addr(), Options{}, 3*time.Second)
		ch <- res{c, err}
	}()
	first := s.read(2 * time.Second)
	if first == nil || first[7] != TypeReliable {
		s.t.Fatalf("no handshake packet: %x", first)
	}
	s.send(0, []byte{TypeControl, ControlAck, first[8], first[9]})
	s.send(0, reliable(seqInitial, []byte{TypeControl, ControlSetPeer, 0, byte(s.id)}))
	r := <-ch
	if r.err != nil {
		s.t.Fatal(r.err)
	}
	return r.c
}

func recvMsg(t *testing.T, c *Conn) Message {
	select {
	case m := <-c.Messages():
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a message")
	}
	return Message{}
}

func TestReliableOrderingAndSplit(t *testing.T) {
	s := newFakeServer(t)
	c := s.handshake()
	defer c.Close()
	if c.PeerID() != 77 {
		t.Fatalf("peer id %d", c.PeerID())
	}

	big := bytes.Repeat([]byte("0123456789"), 120) // forces a split
	// seq 65501: split chunk 2/2, 65500+? deliver out of order:
	// next expected incoming seq is 65501 (65500 was the SET_PEER_ID packet)
	s.send(0, reliable(65502, original([]byte{0, 2, 'B'})))
	s.send(0, reliable(65503, split(1, 2, 1, big[600:])))
	s.send(0, reliable(65501, original([]byte{0, 2, 'A'})))
	s.send(0, reliable(65504, split(1, 2, 0, big[:600]))) // completes split 1 (chunks arrive 1 then 0)

	m1, m2 := recvMsg(t, c), recvMsg(t, c)
	if string(m1.Payload) != "\x00\x02A" || string(m2.Payload) != "\x00\x02B" {
		t.Fatalf("out-of-order delivery: %q %q", m1.Payload, m2.Payload)
	}
	m3 := recvMsg(t, c)
	if !bytes.Equal(m3.Payload, big) {
		t.Fatalf("split reassembly mismatch: %d bytes", len(m3.Payload))
	}

	// every reliable packet must have been acked, including buffered future ones
	acked := map[uint16]bool{}
	deadline := time.Now().Add(time.Second)
	for !(acked[65501] && acked[65502] && acked[65503] && acked[65504]) && time.Now().Before(deadline) {
		d := s.read(200 * time.Millisecond)
		if len(d) >= 11 && d[7] == TypeControl && d[8] == ControlAck {
			acked[binary.BigEndian.Uint16(d[9:])] = true
		}
	}
	for _, seq := range []uint16{65501, 65502, 65503, 65504} {
		if !acked[seq] {
			t.Errorf("seq %d not acked (got %v)", seq, acked)
		}
	}
}

func TestSendSplitsLargePayloadAndResends(t *testing.T) {
	s := newFakeServer(t)
	c := s.handshake()
	defer c.Close()

	payload := bytes.Repeat([]byte{0xab}, 1500)
	if err := c.Send(1, true, payload); err != nil {
		t.Fatal(err)
	}
	// collect reliable split chunks; ignore the first transmission of one
	// chunk to force a retransmission
	got := map[uint16][]byte{}
	dropped := false
	deadline := time.Now().Add(4 * time.Second)
	var count int
	for len(got) < 4 || count == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("timed out; have %d chunks", len(got))
		}
		d := s.read(300 * time.Millisecond)
		if len(d) < 8+7 || d[6] != 1 || d[7] != TypeReliable || d[10] != TypeSplit {
			continue
		}
		seq := binary.BigEndian.Uint16(d[8:])
		count = int(binary.BigEndian.Uint16(d[13:]))
		num := binary.BigEndian.Uint16(d[15:])
		if num == 1 && !dropped {
			dropped = true
			continue // pretend it was lost
		}
		got[num] = append([]byte(nil), d[17:]...)
		s.send(1, []byte{TypeControl, ControlAck, byte(seq >> 8), byte(seq)})
		if len(got) == count {
			break
		}
	}
	var all []byte
	for i := 0; i < count; i++ {
		all = append(all, got[uint16(i)]...)
	}
	if !bytes.Equal(all, payload) {
		t.Fatalf("reassembled %d bytes, want %d", len(all), len(payload))
	}
	if !dropped {
		t.Fatal("test did not exercise retransmission")
	}
}

func TestPingKeepsAlive(t *testing.T) {
	// a DISCO from the server ends the connection
	s := newFakeServer(t)
	c := s.handshake()
	s.send(0, []byte{TypeControl, ControlDisco})
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("connection did not end after DISCO")
	}
	if err := c.Send(0, true, []byte{0, 1}); err == nil {
		t.Fatal("Send on closed connection should fail")
	}
}
