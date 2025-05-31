# CLAUDE.md

## 📸 Screenshots
Screenshots are located in: `~/Dropbox/Screenshots`
Filename format: `YYYY-MM-DD` (e.g., `2025-05-31`)

## 🔰 Objective
Build a production-ready Go MCP server (**ufo-mcp-go**) that:

1. Exposes the **12 capabilities** defined in *MCP_UFO_PLAN.md*  
2. Persists user-defined effects in `/data/effects.json`  
3. Maintains in-memory **shadow LED state** and exposes `getLedState` + `ring_update` events  
4. Ships as a static Docker image (< 20 MB)

---

## 📐 Design Guard-Rails
* ≤ 20 tools.  
* Rich NL descriptions & examples for every tool.  
* `playEffect` & `stopEffects` must stream progress.  
* Protect shared state with `sync.RWMutex` (race-free).  
* Validate all inputs, surface MCP errors on timeouts.

---

## 🛠 Coding Checklist

- [ ] `go mod init github.com/starspace46/ufo-mcp-go`
- [ ] `internal/device` (HTTP wrapper, retries)
- [ ] `internal/effects` (load/save/CRUD, mutex)
- [ ] **NEW `internal/state`** (LedState struct, `Update`, `Snapshot`, event emit)
- [ ] Event bus & SSE fan-out
- [ ] Register tools/resources (`sendRawApi`, `getLedState`, etc.)
- [ ] Unit tests: CRUD, ring pattern, state updates, SSE
- [ ] `make docker` builds image

---

## 🧪 Testing

* Use `httptest` + `UFO_IP=http://localhost:8000` for device calls  
* Cover `getLedState` correctness and `ring_update` emission  
* Integration test: run server, subscribe SSE, call `setRingPattern`, assert `ring_update`

---

## ✔️ Definition of Done

* All checklist boxes ticked (including shadow-state tasks)
* `go test ./... -race` passes
* `curl …/getLedState` returns expected JSON after any mutating tool
* SSE subscriber receives `ring_update` events