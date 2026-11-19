// Command fetchall connects to a server and collects everything it sends
// while joining: the node definitions, map blocks and media files. All of it
// is kept in memory (no files are written); a summary is printed at the end.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"luanti-client/client"
	"luanti-client/mapdump"
	"luanti-client/protocol"
)

func main() {
	var (
		mu       sync.Mutex
		serVer   uint8                         // map block serialization version
		blocks   = map[protocol.V3S16][]byte{} // raw serialized map blocks by position
		media    = map[string][]byte{}         // media file name -> content
		announce = map[string]bool{}           // media names the server announced
		nodedefs = make(chan struct{})         // closed once the node definitions arrived
		closeOne sync.Once
	)

	c := client.New(client.Config{
		Address:      "localhost:30000",
		Username:     "bot",
		Password:     os.Getenv("LUANTI_PASSWORD"),
		Register:     true,
		RequestMedia: true, // ask for every announced media file
		ViewRange:    8,    // map blocks (16 nodes each) the server should send
		Logf:         log.Printf,
	})

	// Handlers must be registered before Connect. They run on the receive
	// loop, so keep them short.
	client.Handle(c, func(p *protocol.ToClientHello) {
		mu.Lock()
		serVer = p.SerializationVersion
		mu.Unlock()
	})
	client.Handle(c, func(*protocol.ToClientNodeDef) {
		closeOne.Do(func() { close(nodedefs) }) // decoded form: c.NodeDefs()
	})
	client.Handle(c, func(p *protocol.ToClientBlockData) {
		mu.Lock()
		blocks[p.Pos] = append([]byte(nil), p.Data...) // a block may be sent again
		mu.Unlock()
	})
	client.Handle(c, func(p *protocol.ToClientAnnounceMedia) {
		mu.Lock()
		for _, f := range p.Files {
			announce[f.Name] = true
		}
		mu.Unlock()
	})
	client.Handle(c, func(p *protocol.ToClientMedia) {
		mu.Lock()
		for _, f := range p.Files {
			media[f.Name] = f.Data
		}
		mu.Unlock()
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := c.Connect(ctx); err != nil { // returns once joined
		log.Fatal(err)
	}
	defer c.Close()

	// Node definitions arrive right after login, before the join completes.
	select {
	case <-nodedefs:
	case <-time.After(10 * time.Second):
		log.Fatal("no node definitions received")
	case <-ctx.Done():
		return
	}
	defs := c.NodeDefs()

	// Map blocks and media keep streaming in; let them arrive for a while.
	select {
	case <-time.After(10 * time.Second):
	case <-c.Done():
	case <-ctx.Done():
	}

	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("%d node definitions, e.g. %q\n", len(defs), defs[0].Name)
	fmt.Printf("%d of %d announced media files\n", len(media), len(announce))

	// Map blocks arrive in the server's serialization format with global
	// content ids; mapdump decodes them and resolves the ids to node names.
	names := mapdump.Names(defs)
	for pos, data := range blocks {
		mb, err := mapdump.Convert(serVer, data, names)
		if err != nil {
			log.Printf("block %v: %v", pos, err)
			continue
		}
		fmt.Printf("block (%d,%d,%d): %d distinct nodes\n", pos.X, pos.Y, pos.Z, len(mb.NodeMapping))
	}
}
