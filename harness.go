package goserve

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// DefaultBin is where the llama.cpp unified binary is expected by convention
// (`~/.llama-app/llama`). Override it in Config.Bin.
func DefaultBin() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".llama-app", "llama")
}

// Model is one server to bring up: a GGUF file under Config.Dir, a port, and a
// context window. Embed selects the embeddings server (otherwise a chat server).
type Model struct {
	Name  string
	File  string // GGUF filename under Config.Dir
	Port  int
	Ctx   int
	Embed bool
}

// Config describes a local model stack. Nothing is hardcoded: an app names its
// own models, ports, and paths, which is what makes the live gate drop-in.
type Config struct {
	Bin          string // llama binary; "" uses DefaultBin()
	Dir          string // directory holding the GGUF files; "" uses "."
	Host         string // bind host; "" uses 127.0.0.1
	Models       []Model
	Reclaim      bool          // kill anything on the model ports before starting (test harnesses)
	ReadyTimeout time.Duration // per-server readiness wait; 0 uses 180s
	Stdout       io.Writer     // child log sink; nil uses os.Stdout
}

func (c Config) bin() string {
	if c.Bin != "" {
		return c.Bin
	}
	return DefaultBin()
}
func (c Config) dir() string {
	if c.Dir != "" {
		return c.Dir
	}
	return "."
}
func (c Config) host() string {
	if c.Host != "" {
		return c.Host
	}
	return "127.0.0.1"
}
func (c Config) readyTimeout() time.Duration {
	if c.ReadyTimeout > 0 {
		return c.ReadyTimeout
	}
	return 180 * time.Second
}

// Available reports whether the llama binary and every configured GGUF exists,
// so a caller can fail with a clear message (rather than a cryptic start error)
// when the lab is not set up.
func (c Config) Available() (bool, string) {
	paths := []string{c.bin()}
	for _, m := range c.Models {
		paths = append(paths, filepath.Join(c.dir(), m.File))
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return false, fmt.Sprintf("local model not available: %s missing", p)
		}
	}
	return true, ""
}

// Harness owns the running model servers for one session.
type Harness struct {
	sup  *Supervisor
	cfg  Config
	host string
	port map[string]int
}

// Start reclaims the ports (if asked), launches every configured model server,
// and waits for each to be genuinely ready (a generated token). The caller must
// Close the returned Harness. It does not start any gateway: point your own
// gateway or client at the URLs the Harness exposes.
func Start(ctx context.Context, cfg Config) (*Harness, error) {
	if ok, msg := cfg.Available(); !ok {
		return nil, fmt.Errorf("goserve: %s", msg)
	}
	logw := cfg.Stdout
	if logw == nil {
		logw = os.Stdout
	}
	if cfg.Reclaim {
		for _, m := range cfg.Models {
			ReclaimPort(m.Port, logw)
		}
	}
	h := &Harness{cfg: cfg, host: cfg.host(), port: map[string]int{}}
	var servers []Server
	for _, m := range cfg.Models {
		path := filepath.Join(cfg.dir(), m.File)
		if m.Embed {
			servers = append(servers, EmbedServer(m.Name, cfg.bin(), cfg.host(), path, m.Port, m.Ctx))
		} else {
			servers = append(servers, LlamaServer(m.Name, cfg.bin(), cfg.host(), path, m.Port, m.Ctx))
		}
		h.port[m.Name] = m.Port
	}
	sup := &Supervisor{Servers: servers, Stdout: logw}
	if err := sup.Start(ctx); err != nil {
		return nil, err
	}
	if err := sup.WaitHealthy(ctx, cfg.readyTimeout()); err != nil {
		sup.Shutdown()
		return nil, err
	}
	h.sup = sup
	return h, nil
}

// BaseURL returns http://host:port for the named model, or "" if unknown.
func (h *Harness) BaseURL(name string) string {
	p, ok := h.port[name]
	if !ok {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", h.host, p)
}

// ChatURL returns the OpenAI-compatible chat-completions endpoint for the named
// model, the usual Target for a gorauder attack.
func (h *Harness) ChatURL(name string) string {
	if b := h.BaseURL(name); b != "" {
		return b + "/v1/chat/completions"
	}
	return ""
}

// Close stops every model server. Safe to call once.
func (h *Harness) Close() {
	if h.sup != nil {
		h.sup.Shutdown()
	}
}
