package client

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/luanti-golang/luanti-client/protocol"
)

// TestLive runs against a real server, e.g.
//
//	docker compose up -d
//	LUANTI_ADDR=localhost:30000 go test ./client -run Live -v
func TestLive(t *testing.T) {
	addr := os.Getenv("LUANTI_ADDR")
	if addr == "" {
		t.Skip("set LUANTI_ADDR to run against a live server")
	}
	name := "gotest" + time.Now().Format("0102150405")
	seen := map[string]int{}
	c := New(Config{
		Address: addr, Username: name, Password: "s3cret-pw", Register: true,
		Tap: func(ev Event) {
			seen[protocol.Name(ev.Packet)]++
			if ev.Err != nil {
				t.Errorf("decode error: %v", ev.Err)
			}
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	c.Close()
	found := false
	for _, n := range c.NodeDefs() {
		if n.Name == "default:stone" {
			found = n.Groups["cracky"] == 3 && n.DrawType == "normal"
		}
	}
	if !found {
		t.Error("default:stone missing or wrong in decoded node definitions")
	}
	for _, want := range []string{"TOCLIENT_HELLO", "TOCLIENT_AUTH_ACCEPT", "TOCLIENT_NODEDEF", "TOCLIENT_PRIVILEGES"} {
		if seen[want] == 0 {
			t.Errorf("never saw %s", want)
		}
	}

	// a second login with the same name must use SRP and the right password
	wrong := New(Config{Address: addr, Username: name, Password: "bad"})
	if err := wrong.Connect(ctx); err == nil {
		t.Fatal("wrong password accepted")
	}
	// and without Register an unknown name is refused
	unknown := New(Config{Address: addr, Username: name + "x", Password: "pw"})
	if err := unknown.Connect(ctx); err != ErrNotRegistered {
		t.Fatalf("got %v, want ErrNotRegistered", err)
	}
}
