# **MCP_UFO_PLAN.md**







## **0 Overview**





A Go-based **Model Context Protocol (MCP) server** that exposes the Dynatrace UFO’s lighting API as LLM-friendly tools, user-editable lighting effects, and real-time state events. It lives in a Docker container and talks to the UFO over its native REST /api.



- **Language/runtime:** Go 1.22, static build
- **MCP helper lib:** github.com/mark3labs/mcp-go/server
- **Container base:** distroless static
- **Persistence:** effects.json in /data
- **Network:** UFO reachable at $UFO_IP (env var) via HTTP





------





## **1 Capabilities Exposed**



| **Name**       | **Kind** | **Description**                                              | **Key Params**                                      | **Streaming** |
| -------------- | -------- | ------------------------------------------------------------ | --------------------------------------------------- | ------------- |
| sendRawApi     | Tool     | Fire a raw query string exactly as typed in UFO web UI.      | query (string)                                      | ❌             |
| setRingPattern | Tool     | High-level wrapper around *_init, *_bg, *_whirl, *_morph.    | ring, segments[], background?, whirlMs?, morphSpec? | ❌             |
| setLogo        | Tool     | Turn Dynatrace logo LED on/off.                              | state (“on”/“off”)                                  | ❌             |
| setBrightness  | Tool     | Global dim 0-255.                                            | level (int)                                         | ❌             |
| playEffect     | Tool     | Run built-in or user-defined effect for *n* seconds; progress streamed every sec. | name, seconds?=10                                   | ✅             |
| stopEffects    | Tool     | Cancel any running effect; instant progress/ack.             | –                                                   | ✅             |
| addEffect      | Tool     | Add a new effect to catalog.                                 | effect (object)                                     | ❌             |
| updateEffect   | Tool     | Modify existing effect.                                      | effect (object)                                     | ❌             |
| deleteEffect   | Tool     | Remove effect by name.                                       | name                                                | ❌             |
| listEffects    | Tool     | Return catalog array.                                        | –                                                   | ❌             |
| getStatus      | Resource | JSON with ssid,ip,firmware,uptime,dimLevel,runningEffect.    | –                                                   | –             |
| stateEvents    | Stream   | Pushes effect_started, effect_stopped, dim_changed, button_press, raw_executed. | –                                                   | SSE           |

Total = **11** capabilities (well under Anthropic’s recommended <20).



------





## **2 Effect Object Schema**



```
{
  "name":        "string (unique)",
  "description": "string",
  "pattern":     "string  // raw /api query WITHOUT leading '?'/path",
  "duration":    10       // default run time seconds
}
```

Stored in /data/effects.json and guarded by sync.RWMutex for multi-client safety.



------





## **3 Server Architecture**





*Main goroutine*



1. Initialise MCP server (mcp.NewServer("dynatrace-ufo")).
2. Register tools/resources per table above.
3. Start **event bus** (chan Event) + SSE fan-out.
4. Launch **button poller** (optional) hitting /api?button_state=1.
5. ListenAndServe(":8080") – exposes JSON-RPC + SSE at /.well-known/mcp/*.





*Effect Runner*: playEffect spins a goroutine → ticks progress → stops.



------





## **4 Build & Run**



```
# build
GOOS=linux  CGO_ENABLED=0  go build -o mcp-ufo

# container
FROM gcr.io/distroless/static
COPY mcp-ufo /mcp-ufo
EXPOSE 8080
ENV UFO_IP=ufo
ENTRYPOINT ["/mcp-ufo"]
# run (with persistence)
docker run -d \
  -p 8080:8080 \
  -e UFO_IP=10.1.23.116 \
  -v $(pwd)/data:/data \
  --name ufo-mcp-go ufo-mcp-go
```



------





## **5 Initial Seed Effects**



| **name**       | **description**             | **pattern**                     | **duration** |
| -------------- | --------------------------- | ------------------------------- | ------------ |
| rainbow        | Slow moving rainbow         | effect=rainbow                  | 15           |
| policeLights   | Alternating red/blue flash  | *big query string*              | 20           |
| breathingGreen | Fade in/out green           | `top_init=1&bottom_init=1&top=0 | 15           |
| pipelineDemo   | Blog demo two-stage colours | custom                          | 10           |
| ipDisplay      | Spell IP address            | effect=ip                       | 30           |



------





## **6 Future Enhancements**





- Auth/API keys per client.
- Rate-limit wrapper (golang.org/x/time/rate).
- Web dashboard via HTMX consuming the same SSE.
- Auto-detect UFO IP via mDNS scan on startup.
