package client

import (
	"fmt"
	"path"
	"strings"

	"github.com/luanti-golang/luanti-client/protocol"
)

// Filter selects which packets are reported to the tap (and thereby dumped).
//
// A filter is a comma-separated list of rules:
//
//	chat_message          include packets by name, without the TOCLIENT_/TOSERVER_ prefix
//	TOCLIENT_HP           ...or with it
//	*_SOUND, hud*         shell-style wildcards (case-insensitive)
//	in:time_of_day        only server->client packets (alias: s2c:)
//	out:*                 only client->server packets (alias: c2s:)
//	!blockdata            exclude (also "-blockdata"); excludes always win
//	all                   everything (the default for an empty filter)
//
// If there is at least one include rule, a packet must match one of them.
type Filter struct {
	rules []filterRule
	hasIn bool
}

type filterRule struct {
	exclude bool
	dirs    [2]bool // indexed by protocol.Direction: applies to ToClient, ToServer
	pattern string  // upper case, no TOCLIENT_/TOSERVER_ prefix
}

// ParseFilter parses a filter specification. An empty spec matches everything.
func ParseFilter(spec string) (*Filter, error) {
	f := &Filter{}
	for _, tok := range strings.Split(spec, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		r := filterRule{dirs: [2]bool{true, true}}
		if tok[0] == '!' || tok[0] == '-' {
			r.exclude = true
			tok = strings.TrimSpace(tok[1:])
		}
		if i := strings.IndexByte(tok, ':'); i >= 0 {
			switch strings.ToLower(tok[:i]) {
			case "in", "s2c", "toclient":
				r.dirs = [2]bool{true, false}
			case "out", "c2s", "toserver":
				r.dirs = [2]bool{false, true}
			default:
				return nil, fmt.Errorf("filter: unknown direction %q in %q (use in: or out:)", tok[:i], tok)
			}
			tok = tok[i+1:]
		}
		p := strings.ToUpper(strings.TrimSpace(tok))
		p = strings.TrimPrefix(strings.TrimPrefix(p, "TOCLIENT_"), "TOSERVER_")
		if p == "" {
			return nil, fmt.Errorf("filter: empty pattern in %q", tok)
		}
		if _, err := path.Match(p, ""); err != nil {
			return nil, fmt.Errorf("filter: bad pattern %q: %v", tok, err)
		}
		if p == "ALL" {
			p = "*"
		}
		r.pattern = p
		if !r.exclude {
			f.hasIn = true
		}
		f.rules = append(f.rules, r)
	}
	return f, nil
}

// Match reports whether a packet passes the filter. A nil Filter matches everything.
func (f *Filter) Match(dir protocol.Direction, p protocol.Packet) bool {
	if f == nil || len(f.rules) == 0 {
		return true
	}
	name := protocol.CommandName(dir, p.Command())
	name = strings.TrimPrefix(strings.TrimPrefix(name, "TOCLIENT_"), "TOSERVER_")
	included := !f.hasIn
	for _, r := range f.rules {
		if !r.dirs[dir] {
			continue
		}
		if ok, _ := path.Match(r.pattern, name); !ok {
			continue
		}
		if r.exclude {
			return false
		}
		included = true
	}
	return included
}
