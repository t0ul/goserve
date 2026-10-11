# goserve

Bring up local llama.cpp model servers with one call, so an app (or its security
gate) can run against a **real** model instead of a stub. Pure Go standard
library, deterministic teardown, scriptable in a test.

Part of the fleet of build-your-own Go security tools. goserve is the live-model
piece: it pairs with [`gorauder`](https://github.com/t0ul/gorauder) (the attack
harness) so the same red-team suite that runs offline can run against the model
itself, every time, locally.

## Why

A test suite that stubs the model is not testing the thing that can be attacked.
The models run locally and start in seconds, so there is no reason the gate should
not include them. goserve makes that drop-in: name your models, ports, and GGUF
directory, and it boots them, waits for genuine generation readiness, and tears
them down cleanly.

## Use

```go
h, err := goserve.Start(ctx, goserve.Config{
    Dir:     "set-up/vm-assets",
    Reclaim: true, // kill anything on the ports first (test harnesses)
    Models: []goserve.Model{
        {Name: "planner", File: "llama-3.2-3b.gguf", Port: 11435, Ctx: 8192},
        {Name: "coder", File: "qwen1.5b.gguf", Port: 11436, Ctx: 4096},
    },
})
if err != nil {
    log.Fatal(err)
}
defer h.Close()

// Point a gorauder Target (or your gateway) at the running model:
url := h.ChatURL("planner") // http://127.0.0.1:11435/v1/chat/completions
```

- `Config.Available()` reports a clear message when the binary or a GGUF is
  missing, so a live test can fail loudly rather than start cryptically.
- `Bin` defaults to `~/.llama-app/llama` (the unified llama.cpp CLI); override it.
- Readiness means a generated token, not merely an open socket.

The binary is expected to accept `serve -m <file> --host --port --ctx-size`
(the llama.cpp unified CLI). Chat and embedding servers are both supported.
