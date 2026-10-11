package goserve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/t0ul/goserve"
)

func writeFile(path string) error { return os.WriteFile(path, []byte("x"), 0o644) }

func TestAvailableMissingReportsPath(t *testing.T) {
	cfg := goserve.Config{
		Bin: "/nonexistent/llama",
		Dir: t.TempDir(),
		Models: []goserve.Model{
			{Name: "planner", File: "nope.gguf", Port: 11999, Ctx: 2048},
		},
	}
	ok, msg := cfg.Available()
	if ok {
		t.Fatal("expected Available=false when bin/model are missing")
	}
	if !strings.Contains(msg, "/nonexistent/llama") {
		t.Fatalf("message should name the missing path, got %q", msg)
	}
}

func TestAvailableTrueWhenPresent(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "llama")
	if err := writeFile(bin); err != nil {
		t.Fatal(err)
	}
	gguf := filepath.Join(dir, "m.gguf")
	if err := writeFile(gguf); err != nil {
		t.Fatal(err)
	}
	cfg := goserve.Config{Bin: bin, Dir: dir, Models: []goserve.Model{{Name: "planner", File: "m.gguf", Port: 1, Ctx: 1}}}
	if ok, msg := cfg.Available(); !ok {
		t.Fatalf("expected Available=true, got %q", msg)
	}
}

func TestDefaultBinEndsWithConvention(t *testing.T) {
	if got := goserve.DefaultBin(); !strings.HasSuffix(got, filepath.Join(".llama-app", "llama")) {
		t.Fatalf("DefaultBin = %q, want it to end with .llama-app/llama", got)
	}
}
