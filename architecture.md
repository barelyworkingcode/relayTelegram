# Architecture

Telegram bot that bridges messages to relayLLM sessions over HTTP.

```
Telegram User ──> Bot (long poll) ──> relayLLM HTTP API ──> LLM
                                                              │
Telegram User <── Bot (reply)     <── relayLLM response  <───┘
```

## relayLLM API Endpoints

| Method | Path | Body | Response |
|--------|------|------|----------|
| `GET` | `/api/projects` | - | `[{id, name, path, model, disabled}]` |
| `POST` | `/api/sessions` | `{projectId, name?, callbackType?}` | `{sessionId, projectId, model}` |
| `POST` | `/api/sessions/:id/message` | `{text}` | `{response, stats, error?}` |

## Message Flow

1. Telegram long poll receives message
2. Auth guard checks sender ID against `TELEGRAM_ALLOWED_USER_ID`
3. Lookup chat mapping to find linked project
4. Lookup session mapping for chat+thread; if none, `POST /api/sessions` to create one (with `callbackType: "policy"`)
5. `POST /api/sessions/:id/message` with the user's text
6. **Block** waiting for the full HTTP response (relayLLM completes entire LLM generation before responding)
7. Split response at 4096-char Telegram limit (paragraph boundaries preferred) and reply

This is a synchronous, blocking transaction -- no streaming, callbacks, or async polling. One HTTP request in, one complete response out, then reply to Telegram. A background goroutine sends Telegram typing indicators every 5 seconds while the call blocks so the user sees activity in their chat.

## Session Lifecycle

- **Created** on first message per chat+thread combination via `POST /api/sessions`
- **Forum topics** get separate sessions (keyed by thread ID); regular chats use a single `"default"` session
- **Session names**: forum topics are named `"Telegram thread <id>"`; default sessions have no name
- **`/clear`** forwards to relayLLM (resets server-side context); local mapping is preserved
- **Session expiry**: if relayLLM returns an error containing `"not found"`, the local mapping is cleared and the user is prompted to resend

## Command Routing

**Local only** (handled by bot, never reach relayLLM):
`/start`, `/link`, `/unlink`, `/projects`, `/status`

**Forwarded to relayLLM**:
`/clear`, `/compact`, `/model`, and any unrecognized `/command`

**Hybrid**:
`/help` -- replies with bot command list locally, then forwards `/help` to relayLLM if an active session exists (shows session commands)

## Error Handling

| Condition | HTTP Status | Bot behavior |
|-----------|-------------|-------------|
| Session busy | 409 | Reply: "session is busy processing another message" |
| Timeout | 504 | Reply: "response timed out" |
| Session expired | any (error contains "not found") | Clear local mapping, prompt user to resend |
| relayLLM unreachable | - | Reply with connection error |

Client timeout is 6 minutes (exceeds relayLLM's 5-minute timeout so relayLLM always responds first).

## Mapping Persistence

Stored at `~/.config/relay/telegram-mappings.json`:

```json
{
  "chatMappings": {
    "<chatId>": {
      "projectId": "...",
      "projectName": "...",
      "sessions": {
        "<threadId|default>": {
          "sessionId": "...",
          "lastActive": "2025-01-01T00:00:00Z"
        }
      }
    }
  }
}
```

- `chatId` -- string-encoded Telegram chat ID
- `threadId` -- string-encoded forum topic ID, or `"default"` for non-forum chats
- Linking a chat (`/link`) replaces the entire chat mapping (clears all sessions)
- Unlinking (`/unlink`) deletes the chat mapping entirely
