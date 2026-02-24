# browser-actions examples

These examples are intended to be runnable from the `browser-actions/` directory.

## Quick start

```bash
go run ./cmd/browser-actions validate examples/basic-navigation.yaml
go run ./cmd/browser-actions run examples/basic-navigation.yaml
```

## Example files

- `basic-navigation.yaml`
  - Minimal script: navigate, wait for load, assert text, screenshot.
- `dom-playground.yaml`
  - Rich demo that injects a small DOM playground into `https://example.com` and exercises:
    `evaluate`, `click`, `fill`, `wait`, `assert`, `scroll`, `sleep`, and `screenshot`.
- `page-scroll.json`
  - JSON object example that demonstrates page-level scroll without a selector and evaluation.
- `raw-array.json`
  - JSON array input (instead of `{ title, actions }`) to demonstrate the alternate parser path.
- `invalid-demo.yaml`
  - Intentionally invalid script for testing `validate` diagnostics.
- `offline-local.yaml`
  - Full success demo against the local fixture server (no external network required).
- `fixtures/index.html`
  - Static page served by the local fixture server for offline demos.
- `fixture-server/main.go`
  - Tiny local HTTP server that serves `examples/fixtures`.

## Useful commands

```bash
# Human-readable output
go run ./cmd/browser-actions run examples/dom-playground.yaml

# JSON output
go run ./cmd/browser-actions run --json examples/dom-playground.yaml

# Validate using the compatibility profile
go run ./cmd/browser-actions validate --profile sukyan-legacy examples/raw-array.json
```

## Offline demo (no external network)

From `browser-actions/`:

```bash
# Terminal 1
go run ./examples/fixture-server

# Terminal 2
go run ./cmd/browser-actions run examples/offline-local.yaml
```

## Notes

- Some examples use `https://example.com` as a stable target and inject their own test DOM where needed.
- The offline example uses a local fixture server and requires no external network.
- Screenshot files are written under `examples/output/` by default based on the paths in the scripts.
