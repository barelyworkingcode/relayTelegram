# relayTelegram

Telegram bot that bridges messages to [Eve](../eve/) LLM sessions. Managed as a background service by [Relay](../relay/).

## Setup

1. Create a bot with [@BotFather](https://t.me/BotFather) on Telegram
2. Get your numeric user ID (message [@userinfobot](https://t.me/userinfobot))
3. Build and register:
   ```bash
   ./build.sh
   ```
4. Open Relay settings and set environment variables:
   - `TELEGRAM_BOT_TOKEN` - Token from BotFather
   - `TELEGRAM_ALLOWED_USER_ID` - Your numeric Telegram user ID
   - `EVE_URL` - Eve server URL (default: `http://localhost:3000`)

## Usage

- `/start` - Health check (shows Eve connectivity for allowed user, dismisses others)
- `/help` - Show bot commands and Eve session commands
- `/link <projectName>` - Link chat to an Eve project
- `/unlink` - Remove link
- `/projects` - List available Eve projects
- `/status` - Show current mapping
- `/clear` - Start new session in this thread

Messages in a linked chat are sent to Eve and responses returned. Forum topics get separate sessions; DMs use a single session per chat.

Unrecognized `/` commands (e.g. `/compact`, `/model`, `/cost`) are forwarded to Eve as session commands.

## Attribution

Telegram Bot API client: [telebot](https://gopkg.in/telebot.v4) by [tucnak](https://github.com/tucnak/telebot), MIT License.
