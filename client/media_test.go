package client

import (
	"crypto/sha1"
	"os"
	"path/filepath"
	"testing"

	"github.com/luanti-golang/luanti-client/protocol"
)

func TestSaveMedia(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "media")
	var warnings []string
	c := New(Config{MediaDir: dir, Logf: func(f string, a ...any) { warnings = append(warnings, f) }})
	good, bad := []byte("png-data"), []byte("tampered")
	c.announced = map[string][20]byte{
		"good.png": sha1.Sum(good),
		"bad.png":  sha1.Sum([]byte("something else")),
		"../evil":  sha1.Sum(good),
	}
	c.saveMedia(&protocol.ToClientMedia{Files: []protocol.MediaFile{
		{Name: "good.png", Data: good},
		{Name: "bad.png", Data: bad},
		{Name: "../evil", Data: good},
		{Name: "unannounced.png", Data: good},
	}})

	if b, err := os.ReadFile(filepath.Join(dir, "good.png")); err != nil || string(b) != "png-data" {
		t.Fatalf("good file not saved: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected only the good file, got %d entries", len(entries))
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "evil")); err == nil {
		t.Fatal("path traversal succeeded")
	}
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d", len(warnings))
	}
	if !c.haveMedia("good.png", sha1.Sum(good)) || c.haveMedia("good.png", sha1.Sum(bad)) {
		t.Fatal("haveMedia wrong")
	}
}
