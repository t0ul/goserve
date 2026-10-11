//go:build live

package goserve_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/t0ul/goserve"
)

// TestLiveBootAndAnswer is the drop-in proof: with the local lab set up, a fresh
// caller boots a chat model through goserve and gets a real answer, in one short
// run. It FAILS (never skips) when the lab is missing, because a gate that skips
// is not a gate. Models are found under GOSERVE_DIR (default set-up/vm-assets).
func TestLiveBootAndAnswer(t *testing.T) {
	dir := os.Getenv("GOSERVE_DIR")
	if dir == "" {
		dir = "set-up/vm-assets"
	}
	cfg := goserve.Config{
		Dir:          dir,
		Reclaim:      true,
		ReadyTimeout: 90 * time.Second,
		Stdout:       io.Discard,
		Models: []goserve.Model{
			{Name: "planner", File: "llama-3.2-3b.gguf", Port: 11435, Ctx: 8192},
		},
	}
	if ok, msg := cfg.Available(); !ok {
		t.Fatalf("live lab not set up (%s); the live gate must not be skipped", msg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	start := time.Now()
	h, err := goserve.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("goserve.Start: %v", err)
	}
	defer h.Close()

	body := []byte(`{"model":"planner","messages":[{"role":"user","content":"Reply with the single word READY."}],"max_tokens":8,"temperature":0}`)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ChatURL("planner"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("chat request: %v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat status %d: %s", resp.StatusCode, out)
	}
	t.Logf("live boot + answer in %s; %d bytes", time.Since(start).Round(time.Second), len(out))
}
