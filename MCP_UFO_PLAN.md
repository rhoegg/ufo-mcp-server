# MCP_UFO_PLAN.md

## 0  Overview
A Go-based **Model Context Protocol (MCP) server** that exposes the Dynatrace UFO’s lighting API as:

* LLM-friendly tools  
* user-editable lighting effects (CRUD)  
* real-time state events  
* **in-memory shadow copy of the current LED state** (Path A)

Everything ships in a Docker container and talks to the UFO over its native `/api`.

* **Language/runtime:** Go 1.23 (static)
* **MCP helper lib:** <https://github.com/mark3labs/mcp-go>
* **Container base:** distroless `static`
* **Persistence:** `effects.json` in `/data`
* **Network:** UFO reachable at `$UFO_IP` (env var) via HTTP
* **Shadow LED state (Path A):** exposed via `getLedState`

---

## 1  Capabilities Exposed

| Name | Kind | Description | Key Params | Streaming |
|------|------|-------------|------------|-----------|
| `sendRawApi` | Tool | Fire a raw query string exactly as typed in the UFO web UI. | `query` (string) | ❌ |
| `setRingPattern` | Tool | Wrapper around `*_init`, `*_bg`, `*_whirl`, `*_morph`. | `ring`, `segments[]`, `background?`, `whirlMs?`, `morphSpec?` | ❌ |
| `setLogo` | Tool | Turn the Dynatrace logo LED on/off. | `state` (`"on"`\|`"off"`) | ❌ |
| `setBrightness` | Tool | Global dim 0-255 (**updates shadow state**). | `level` (int) | ❌ |
| `playEffect` | Tool | Run a built-in or user-defined effect for *n* seconds; streams progress; **updates shadow state**. | `name`, `seconds?=10` | ✅ |
| `stopEffects` | Tool | Cancel any running effect; **updates shadow state**. | – | ✅ |
| `addEffect` / `updateEffect` / `deleteEffect` / `listEffects` | Tools | CRUD the effect catalogue. | – | ❌ |
| **`getLedState`** | Tool | **NEW** – returns JSON shadow state `{top[15], bottom[15], logoOn, effect, dim}`. | – | ❌ |
| `getStatus` | Resource | Wi-Fi SSID, IP, firmware, uptime, brightness, runningEffect. | – | – |
| `ufo://ledstate` | Resource | Shadow LED state (also accessible via getLedState tool). | – | – |
| **`stateEvents`** | Stream | Pushes `effect_started`, `effect_stopped`, `dim_changed`, **`ring_update`**, `button_press`, `raw_executed`. | – | SSE |

**Total = 12 capabilities** (well under Anthropic’s < 20 guideline).

---

## 2  Shadow LED State (Path A)

```go
type LedState struct {
    Top    [15]string // hex colours
    Bottom [15]string
    LogoOn bool
    Effect string
    Dim    int
}