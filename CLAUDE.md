# relayTelegram

Telegram bot bridge to Eve LLM sessions. Go binary managed by Relay.

## Architecture

```
Telegram User -> Bot (long polling) -> Auth check -> Eve HTTP API -> LLM response -> Telegram reply
```

Telegram chats map to Eve projects (`/link`), forum topics map to Eve sessions (auto-created).

## Files

- `main.go` - Entry point, config loading, graceful shutdown
- `bot.go` - Telegram bot, auth guard, message routing, commands. `/start` bypasses auth guard (dismisses unknown users, shows Eve status to allowed user). `/help` shows bot commands then forwards to Eve for session commands. Unrecognized `/` commands pass through to Eve (e.g. `/compact`, `/model`).
- `eve.go` - HTTP client to Eve API (`/api/sessions`, `/api/sessions/:id/message`)
- `mappings.go` - JSON persistence for chat-to-project-to-session mappings

## Env Vars

- `TELEGRAM_BOT_TOKEN` - From BotFather (required)
- `TELEGRAM_ALLOWED_USER_ID` - Numeric Telegram user ID (required)
- `EVE_URL` - Default `http://localhost:3000`

## Build

```bash
./build.sh  # Compiles binary and registers with Relay (first run only)
```

## Dependencies

- `gopkg.in/telebot.v4` (telebot by tucnak) - Telegram Bot API
- Eve running on EVE_URL with `/api/sessions` and `/api/sessions/:id/message` endpoints
