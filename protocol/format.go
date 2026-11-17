package protocol

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// FormatOptions control Format.
type FormatOptions struct {
	// MaxBytes is how many bytes of a byte slice are shown (default 16).
	MaxBytes int
	// MaxString truncates long strings to this many bytes (default 120).
	MaxString int
}

func (o *FormatOptions) defaults() {
	if o.MaxBytes <= 0 {
		o.MaxBytes = 16
	}
	if o.MaxString <= 0 {
		o.MaxString = 120
	}
}

// Format renders a packet on one line, e.g.
//
//	TOCLIENT_TIME_OF_DAY{Time:6000 Speed:72}
//
// Large byte slices and strings are abbreviated.
func Format(p Packet, opts FormatOptions) string {
	opts.defaults()
	var sb strings.Builder
	sb.WriteString(Name(p))
	if u, ok := p.(*Unknown); ok {
		fmt.Fprintf(&sb, "{Cmd:%#x ", u.Cmd)
		formatValue(&sb, reflect.ValueOf(u.Data), opts)
		sb.WriteString("}")
		return sb.String()
	}
	sb.WriteString("{")
	formatStructFields(&sb, reflect.ValueOf(p).Elem(), opts)
	sb.WriteString("}")
	return sb.String()
}

func formatStructFields(sb *strings.Builder, v reflect.Value, opts FormatOptions) {
	first := true
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			if !first {
				sb.WriteString(" ")
			}
			formatStructFields(sb, v.Field(i), opts)
			first = false
			continue
		}
		if !first {
			sb.WriteString(" ")
		}
		first = false
		sb.WriteString(f.Name)
		sb.WriteString(":")
		formatValue(sb, v.Field(i), opts)
	}
}

func formatValue(sb *strings.Builder, v reflect.Value, opts FormatOptions) {
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if len(s) > opts.MaxString {
			fmt.Fprintf(sb, "%q…(%d bytes)", s[:opts.MaxString], len(s))
		} else {
			fmt.Fprintf(sb, "%q", s)
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			var b []byte
			if v.Kind() == reflect.Slice {
				b = v.Bytes()
			} else {
				b = make([]byte, v.Len())
				for i := range b {
					b[i] = byte(v.Index(i).Uint())
				}
			}
			if len(b) > opts.MaxBytes {
				fmt.Fprintf(sb, "%s…(%d bytes)", hex.EncodeToString(b[:opts.MaxBytes]), len(b))
			} else {
				fmt.Fprintf(sb, "%s", hex.EncodeToString(b))
			}
			return
		}
		sb.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				sb.WriteString(" ")
			}
			if i >= 8 {
				fmt.Fprintf(sb, "…(%d items)", v.Len())
				break
			}
			formatValue(sb, v.Index(i), opts)
		}
		sb.WriteString("]")
	case reflect.Struct:
		sb.WriteString("{")
		formatStructFields(sb, v, opts)
		sb.WriteString("}")
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(sb, "%g", v.Float())
	default:
		if s, ok := v.Interface().(fmt.Stringer); ok {
			sb.WriteString(s.String())
			return
		}
		fmt.Fprintf(sb, "%v", v.Interface())
	}
}

// JSON renders a packet as a JSON object with "dir", "name", "command" and
// "packet" members. Byte slices are base64 encoded by encoding/json.
func JSON(p Packet) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(struct {
		Dir     string `json:"dir"`
		Name    string `json:"name"`
		Command uint16 `json:"command"`
		Packet  Packet `json:"packet"`
	}{p.Direction().String(), Name(p), p.Command(), p})
	return bytes.TrimRight(buf.Bytes(), "\n"), err
}
