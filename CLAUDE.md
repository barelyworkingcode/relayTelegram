# relayTelegram

Telegram bot bridge to relayLLM sessions. Go binary managed by Relay.

## Architecture

```
Telegram User -> Bot (long polling) -> Auth check -> relayLLM HTTP API -> LLM response -> Telegram reply
```

Telegram chats map to relayLLM projects (`/link`), forum topics map to sessions (auto-created).

## Files

- `main.go` - Entry point, config loading, graceful shutdown
- `bot.go` - Telegram bot, auth guard, message routing, commands. `/start` bypasses auth guard (dismisses unknown users, shows relayLLM status to allowed user). `/help` shows bot commands then forwards to relayLLM for session commands. Unrecognized `/` commands pass through to relayLLM (e.g. `/compact`, `/model`).
- `llm.go` - HTTP client to relayLLM API (`/api/sessions`, `/api/sessions/:id/message`)
- `mappings.go` - JSON persistence for chat-to-project-to-session mappings

## Env Vars

- `TELEGRAM_BOT_TOKEN` - From BotFather (required)
- `TELEGRAM_ALLOWED_USER_ID` - Numeric Telegram user ID (required)
- `RELAY_LLM_URL` - Default `http://localhost:3001`

## Build

```bash
./build.sh  # Compiles binary and registers with Relay (first run only)
```

## Dependencies

- `gopkg.in/telebot.v4` (telebot by tucnak) - Telegram Bot API
- relayLLM running on RELAY_LLM_URL with `/api/sessions` and `/api/sessions/:id/message` endpoints
