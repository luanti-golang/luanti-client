// Command luanti-client is a headless Luanti client that logs in to a server
// and dumps the packets that go over the wire.
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/luanti-golang/luanti-client/client"
	"github.com/luanti-golang/luanti-client/protocol"
	"github.com/luanti-golang/luanti-client/transport"
)

const usage = `luanti-client - headless Luanti client

Usage:
  luanti-client connect [flags]   connect, log in and dump packets
  luanti-client packets           list all known packet names (for -filter)

Run "luanti-client connect -h" for the connect flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "connect":
		os.Exit(runConnect(os.Args[2:]))
	case "packets":
		listPackets()
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func listPackets() {
	for _, d := range []protocol.Direction{protocol.ToClient, protocol.ToServer} {
		fmt.Printf("%s (%s):\n", map[protocol.Direction]string{protocol.ToClient: "server -> client (in:)", protocol.ToServer: "client -> server (out:)"}[d], d)
		for _, n := range protocol.CommandNames(d) {
			fmt.Println("  " + n)
		}
	}
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func runConnect(args []string) int {
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	addr := fs.String("addr", "localhost:30000", "server address (host:port)")
	user := fs.String("user", "", "player name (required)")
	pass := fs.String("password", os.Getenv("LUANTI_PASSWORD"), "password (default: $LUANTI_PASSWORD)")
	register := fs.Bool("register", false, "create the account if the server does not know the name")
	filterSpec := fs.String("filter", "", `packets to dump, e.g. "chat_message,hp", "!blockdata", "out:*", "all" (see 'packets')`)
	format := fs.String("format", "text", "dump format: text or json")
	raw := fs.Bool("raw", false, "also dump raw UDP datagrams with transport framing")
	showHex := fs.Bool("hex", false, "include the packet bytes in hex (text format)")
	maxBytes := fs.Int("max-bytes", 16, "bytes of binary fields shown in text format")
	duration := fs.Duration("duration", 0, "disconnect after this long (0: run until interrupted)")
	quiet := fs.Bool("quiet", false, "do not dump packets")
	media := fs.Bool("media", false, "download media files announced by the server (without saving them)")
	mediaDir := fs.String("media-dir", "", "save downloaded media files into this directory (implies -media; skips files already there)")
	nodedefs := fs.String("nodedefs", "", "save the server's node definitions as JSON to this file")
	nodedefsFormat := fs.String("nodedefs-format", "protocol", "schema of -nodedefs: protocol (all fields, list by id) or viewer (luanti-viewer's nodedefs.json)")
	mapblockDir := fs.String("mapblock-dir", "", `save received map blocks into this directory as "(x,y,z).json" (luanti-viewer format)`)
	noAck := fs.Bool("no-block-ack", false, "do not acknowledge received map blocks")
	view := fs.Uint("view-range", 0, "wanted view range in map blocks")
	var say multiFlag
	fs.Var(&say, "say", "chat message to send after joining (repeatable)")
	_ = fs.Parse(args)

	if *user == "" {
		fmt.Fprintln(os.Stderr, "error: -user is required")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(os.Stderr, "error: -format must be text or json")
		return 2
	}
	if *nodedefsFormat != "protocol" && *nodedefsFormat != "viewer" {
		fmt.Fprintln(os.Stderr, "error: -nodedefs-format must be protocol or viewer")
		return 2
	}
	filter, err := client.ParseFilter(*filterSpec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	var outMu sync.Mutex
	start := time.Now()
	printLine := func(format string, a ...any) {
		outMu.Lock()
		defer outMu.Unlock()
		fmt.Printf(format+"\n", a...)
	}
	fopts := protocol.FormatOptions{MaxBytes: *maxBytes}

	cfg := client.Config{
		Address:         *addr,
		Username:        *user,
		Password:        *pass,
		Register:        *register,
		Filter:          filter,
		RequestMedia:    *media,
		MediaDir:        *mediaDir,
		NodeDefsFile:    *nodedefs,
		NodeDefsFormat:  *nodedefsFormat,
		MapblockDir:     *mapblockDir,
		Logf:            func(f string, a ...any) { fmt.Fprintf(os.Stderr, "warning: "+f+"\n", a...) },
		DisableBlockAck: *noAck,
		ViewRange:       uint8(*view),
	}
	if !*quiet {
		cfg.Tap = func(ev client.Event) {
			ts := ev.Time.Sub(start).Seconds()
			if *format == "json" {
				b, err := protocol.JSON(ev.Packet)
				if err != nil {
					b = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
				}
				printLine(`{"t":%.3f,"ch":%d,"packet":%s}`, ts, ev.Channel, b)
				return
			}
			line := fmt.Sprintf("%9.3f %s ch%d %s", ts, ev.Dir, ev.Channel, protocol.Format(ev.Packet, fopts))
			if ev.Err != nil {
				line += "  !! " + ev.Err.Error()
			}
			if *showHex {
				h := ev.Raw
				suffix := ""
				if len(h) > 64 {
					h, suffix = h[:64], "…"
				}
				line += "  hex=" + hex.EncodeToString(h) + suffix
			}
			printLine("%s", line)
		}
		if *raw {
			cfg.OnDatagram = func(dir transport.Direction, d []byte) {
				if *format == "json" {
					return
				}
				printLine("%9.3f UDP %s %s", time.Since(start).Seconds(), dir, transport.Describe(d))
			}
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := client.New(cfg)
	if err := c.Connect(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "connected to %s as %s (protocol %d)\n", *addr, *user, c.ProtocolVersion())
	for _, m := range say {
		if err := c.Chat(m); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
	}

	var timeout <-chan time.Time
	if *duration > 0 {
		timeout = time.After(*duration)
	}
	select {
	case <-ctx.Done():
		c.Close()
	case <-timeout:
		c.Close()
	case <-c.Done():
	}
	if err := c.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, "disconnected:", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "disconnected")
	return 0
}
