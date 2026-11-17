// Command chatbot logs in, says hello, prints chat and reacts to damage.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"luanti-client/client"
	"luanti-client/protocol"
)

func main() {
	c := client.New(client.Config{
		Address:  "localhost:30000",
		Username: "bot",
		Password: os.Getenv("LUANTI_PASSWORD"),
		Register: true, // create the account on first use
	})

	// React to packets by type; each packet is a plain struct.
	c.OnChat(func(m *protocol.ToClientChatMessage) {
		fmt.Println("chat:", m.Message)
	})
	client.Handle(c, func(p *protocol.ToClientHP) {
		fmt.Println("hp is now", p.HP)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := c.Connect(ctx); err != nil { // returns once joined
		log.Fatal(err)
	}
	x, y, z := c.Position()
	fmt.Printf("joined at (%.0f, %.0f, %.0f), privileges: %v\n", x, y, z, c.Privileges())

	_ = c.Chat("hello from Go")

	go func() { <-ctx.Done(); c.Close() }()
	if err := c.Wait(); err != nil { // blocks until disconnected
		log.Fatal(err)
	}
}
