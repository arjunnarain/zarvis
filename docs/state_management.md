# State Management in Zarvis

## The trust boundary

```
  Frontend (display only) <── SSE ── Backend (source of truth) ──── Anthropic API
                                            │
                                            └── SQLite (persistence)
```

The frontend never writes state. The user can't tamper with `localStorage` to jump to Mythic. The LLM can't claim a tool it doesn't have.

## The turn lifecycle

1. Frontend POSTs `{ session_id, message }` to `/api/chat`.
2. Backend loads the session row from SQLite.
3. Backend builds a fresh system prompt with `<current_state>` tags injected.
4. Backend calls Anthropic with that prompt plus a tool list **filtered by stage**.
5. Backend streams the response over SSE. Event types: `delta`, `tool_use`, `evolution`, `done`, `error`.
6. XP is awarded via `evolution.Apply(session, events)`:
   - `+1` per prompt
   - `+2` per successful tool call
   - `+5` per chained tool call (≥2 tools in one turn)
   - `+10` per multi-step workflow
7. If XP crosses a threshold, `session.stage` increments, an `evolution` event fires, and the next turn's prompt reflects the new stage.

## Stage thresholds (`backend/internal/evolution/engine.go`)

| Stage | XP to enter |
|---|---|
| 1 Wisp | 0 |
| 2 Cub | 5 |
| 3 Guardian | 25 |
| 4 Mythic | 100 |

## Frontend event contract

```typescript
type ChatEvent =
  | { type: 'delta'; text: string }
  | { type: 'tool_use'; tool: string; display_name: string; id: string }
  | { type: 'evolution'; from: number; to: number; stage_name: string; unlocked: UnlockedTool[] }
  | { type: 'done'; stage: number; xp: number; xp_to_next: number }
  | { type: 'error'; message: string };
```
