package client

import (
	"testing"

	"luanti-client/protocol"
)

func TestFilter(t *testing.T) {
	hp := &protocol.ToClientHP{}
	blk := &protocol.ToClientBlockData{}
	chat := &protocol.ToServerChatMessage{}
	pos := &protocol.ToServerPlayerPos{}
	cases := []struct {
		spec string
		want [4]bool // hp, blockdata, chat(out), playerpos(out)
	}{
		{"", [4]bool{true, true, true, true}},
		{"all", [4]bool{true, true, true, true}},
		{"!blockdata", [4]bool{true, false, true, true}},
		{"hp", [4]bool{true, false, false, false}},
		{"TOCLIENT_HP,out:chat_message", [4]bool{true, false, true, false}},
		{"out:*", [4]bool{false, false, true, true}},
		{"in:*,!hp", [4]bool{false, true, false, false}},
		{"*_MESSAGE", [4]bool{false, false, true, false}},
		{"all,-playerpos", [4]bool{true, true, true, false}},
		{"in:playerpos", [4]bool{false, false, false, false}},
	}
	for _, c := range cases {
		f, err := ParseFilter(c.spec)
		if err != nil {
			t.Fatalf("%q: %v", c.spec, err)
		}
		got := [4]bool{
			f.Match(protocol.ToClient, hp), f.Match(protocol.ToClient, blk),
			f.Match(protocol.ToServer, chat), f.Match(protocol.ToServer, pos),
		}
		if got != c.want {
			t.Errorf("%q: got %v want %v", c.spec, got, c.want)
		}
	}
	if _, err := ParseFilter("bogus:hp"); err == nil {
		t.Error("expected error for unknown direction")
	}
}
