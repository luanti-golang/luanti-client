# luanti-client

A headless [Luanti](https://www.luanti.org) client written in Go. Use it as a
standalone binary that logs in and dumps every packet on the wire, or as a
library to script bots and tests.

* Speaks the real network protocol (version 54, Luanti 5.18): UDP transport
  with reliable delivery and split packets, SRP authentication, and **all 85
  packet types as Go structs**.
* Logs in with a username and password, optionally **registering the account**
  if the server does not know it yet.
* Dumps every packet in both directions, with a **configurable filter**, as
  text or JSON.
* Saves map blocks, node definitions and media in the export format of
  [luanti-viewer](https://github.com/BuckarooBanzay/luanti-viewer).
* Few dependencies: [klauspost/compress](https://github.com/klauspost/compress)
  (Luanti compresses media and definitions with zstd) and
  [mapparser](https://github.com/luanti-golang/mapparser) (map blocks).

## Quick start

The repository ships a Luanti server as a docker compose service (the
`dev/minetest_game` submodule is mounted into it):

```sh
git submodule update --init
docker compose up -d                       # server on udp/30000

go build -o luanti-client ./cmd/luanti-client
./luanti-client connect -user alice -password secret -register
```

Output looks like this (`S->C` is server to client, `C->S` the reverse, `chN`
the transport channel):

```
    0.006 C->S ch1 TOSERVER_INIT{SerializationVersion:29 ... PlayerName:"alice"}
    0.007 S->C ch0 TOCLIENT_HELLO{SerializationVersion:29 ProtocolVersion:54 AuthMethods:4 ...}
    0.008 C->S ch1 TOSERVER_FIRST_SRP{Salt:264e2a1993f6e9a7…(16 bytes) Verifier:7d68c0…(256 bytes) ...}
    0.014 S->C ch0 TOCLIENT_AUTH_ACCEPT{MapSeed:1829161697494463792 SendInterval:0.09 ...}
    ...
    0.055 S->C ch0 TOCLIENT_CHAT_MESSAGE{Type:0 Message:"*** alice joined the game." ...}
```

## CLI

```
luanti-client connect [flags]   connect, log in and dump packets
luanti-client packets           list all packet names usable in -filter
```

| flag | meaning |
|------|---------|
| `-addr host:port` | server address (default `localhost:30000`) |
| `-user NAME` | player name (required) |
| `-password PW` | password (default: `$LUANTI_PASSWORD`, so it need not appear in `ps`) |
| `-register` | create the account if the server does not know the name |
| `-filter SPEC` | which packets to dump (below) |
| `-format text\|json` | dump format; `json` is one object per line |
| `-raw` | additionally dump raw UDP datagrams with MTP framing (seq numbers, acks, split chunks) |
| `-hex` | append the packet bytes in hex (text format) |
| `-max-bytes N` | how many bytes of binary fields to show in text format (default 16) |
| `-nodedefs FILE` | save the server's node definitions (every node type with its groups, tiles, boxes, sounds, ...) as JSON; written right after login, so `-duration 2s` is enough |
| `-nodedefs-format protocol\|viewer` | schema of `-nodedefs`: `protocol` (default) is every decoded field as a list sorted by content id; `viewer` is the `nodedefs.json` of [luanti-viewer](https://github.com/BuckarooBanzay/luanti-viewer) (node name -> definition, node boxes in node units, only what the viewer renders), see package `viewerdef` |
| `-say MSG` | send a chat message after joining (repeatable) |
| `-duration D` | disconnect after `D` (e.g. `30s`); default: until Ctrl-C |
| `-quiet` | do not dump anything |
| `-media` | download all announced media files without saving them (off by default) |
| `-media-dir DIR` | save media files (textures, sounds, models) into `DIR`; implies `-media`. Files are checked against the announced SHA-1, and ones already in `DIR` are not downloaded again |
| `-mapblock-dir DIR` | save every received map block into `DIR` as `(x,y,z).json` in the mapblock format of [luanti-viewer](https://github.com/BuckarooBanzay/luanti-viewer) (see package `mapdump`); a block the server sends again replaces its file |
| `-view-range N` | wanted view range in map blocks; `0` (default) requests little map data |
| `-no-block-ack` | do not acknowledge map blocks |

Without `-register`, logging in with an unknown name fails with a clear error
instead of silently creating an account. With `-register`, an existing account
is simply logged in to.

### Exporting for luanti-viewer

`-mapblock-dir`, `-nodedefs-format viewer` and `-media-dir` together write
the directory layout luanti-viewer loads (`mapblocks/`, `nodedefs.json`,
`media/`). The server sends the blocks around the player, as far as
`-view-range` asks:

```sh
./luanti-client connect -user alice -password secret -quiet \
    -view-range 6 -duration 20s \
    -mapblock-dir export/mapblocks \
    -nodedefs export/nodedefs.json -nodedefs-format viewer \
    -media-dir export/media
```

Map blocks are parsed with
[mapparser](https://github.com/luanti-golang/mapparser). Over the network
they carry the server's global content ids, so the node names come from the
node definitions. Node metadata (inventories, fields) is not part of the
viewer format and is dropped. Single-node updates (`TOCLIENT_ADDNODE`,
`TOCLIENT_REMOVENODE`) are not applied to the files.

### Filters

A filter is a comma-separated list of rules, applied to packet names without the
`TOCLIENT_`/`TOSERVER_` prefix (case-insensitive, `*` wildcards):

```sh
-filter chat_message,hp          # only these packets
-filter '!blockdata'             # everything except map blocks ("-" also works inside a list: all,-playerpos)
-filter 'out:*'                  # only what the client sends   (in: = what the server sends)
-filter 'hud*,!hudrm,out:chat*'  # wildcards, exclusions and directions combine
```

Exclusions always win; if there is at least one include rule, a packet must
match one. An empty filter, or `all`, dumps everything. Map blocks are chatty,
so `-filter '!blockdata,!active_object*'` is a good start for exploring.

```sh
./luanti-client connect -user alice -password secret \
    -filter 'chat_message,out:chat_message' -say "hello" -duration 10s
```

## Library

The module is named `luanti-client`; import `luanti-client/client` (the helper)
and `luanti-client/protocol` (the packet structs). Rename the module in
`go.mod` to your repository path when you publish it.

```go
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
```

This is [examples/chatbot](examples/chatbot/main.go); run it with
`go run ./examples/chatbot`.

### Helper API

| | |
|---|---|
| `client.New(Config)` | create a client (`Address`, `Username`, `Password`, `Register`, `Tap`, `Filter`, ...) |
| `Connect(ctx)` | dial, authenticate (or register), join; returns once the server sent the initial player state |
| `Wait()` / `Done()` / `Close()` | block until disconnected / channel / disconnect |
| `client.Handle(c, func(*protocol.ToClientX))` | typed handler for one packet type |
| `OnChat`, `OnPacket` | chat handler / handler for every packet in both directions |
| `Send(pkt)` | send any `protocol.ToServer*` packet on the channel Luanti clients use for it |
| `Chat`, `MoveTo`, `Look`, `SetKeys`, `SelectItem`, `Respawn` | common actions |
| `Position`, `HP`, `Privileges`, `Players`, `TimeOfDay`, `MapSeed` | state the client tracks |

Handlers run on the receive loop, so keep them short; they may call `Send`,
`Chat`, etc. `Config.Tap` receives every packet that passes `Config.Filter`
(this is what the CLI uses for its dump), and `Config.OnDatagram` the raw UDP
datagrams. `Connect` and `Wait` return `*client.AccessDeniedError` when the
server refuses or kicks the client (wrong password, banned, shutdown, ...) and
`client.ErrNotRegistered` for an unknown name without `Register`.

Positions are in nodes in the helper API; on the wire they are "BS" units
(10 per node), as in the raw structs.

### Packets as structs

Every packet of the protocol is a struct in `luanti-client/protocol`, named
after the Luanti constant: `TOCLIENT_HP` is `ToClientHP`, `TOSERVER_INTERACT`
is `ToServerInteract`, and so on. They all implement `protocol.Packet`:

```go
pkt, err := protocol.DecodeToClient(payload, 54)   // bytes -> struct
switch p := pkt.(type) {
case *protocol.ToClientTimeOfDay:
	fmt.Println(p.Time, p.Speed)
}
raw, err := protocol.Encode(&protocol.ToServerChatMessage{Message: "hi"}, 54) // struct -> bytes
fmt.Println(protocol.Format(pkt, protocol.FormatOptions{}))                    // one-line text
```

Each struct defines its layout once (a `Sync` method) and gets both decoding and
encoding from it. Layouts that depend on the negotiated protocol version
(HUD sizes, media and definition compression, ...) are handled. Unknown or
malformed packets come back as `*protocol.Unknown` with the raw bytes, so a
dump never loses data.

A few payloads are deliberately kept as raw bytes because they are large
binary formats of their own rather than protocol structure: map blocks
(`ToClientBlockData.Data`), node and item definitions (`ToClientNodeDef`,
`ToClientItemDef`, already decompressed), node metadata, particle parameters
(`ToClientSpawnParticle`, `ToClientAddParticleSpawner`, each entry of
`ToClientSpawnParticleBatch`), and active-object data. Inventories and
formspecs are their textual form.

## Layout

```
cmd/luanti-client/   the CLI
client/              the helper: connect, login/register, handlers, state, filters
protocol/            packet structs, encode/decode, formatting
transport/           UDP transport: peer handshake, channels, reliable + split packets
srp/                 SRP-6a (SHA-256, 2048-bit) as used by Luanti's login
viewerdef/           node definitions -> luanti-viewer nodedefs.json
mapdump/             map blocks -> luanti-viewer mapblock files
examples/chatbot/    the example above
```

## Tests

```sh
go test -race ./...                                  # unit tests, no server needed
LUANTI_ADDR=localhost:30000 go test ./client -run Live -v   # against the docker compose server
```

The unit tests cover encode/decode round trips of every packet, SRP against an
independent server-side implementation, and the transport (out-of-order
reliable delivery, split reassembly, retransmission) against a fake UDP server.

## Scope

This is a protocol client, not a game client: it does not keep a world model
(map blocks are only decoded to save them with `-mapblock-dir`), run
physics, render, or play sounds. By default it does not download media. Map
blocks are acknowledged so that the server keeps sending; use `-view-range`
to ask for more of them.
