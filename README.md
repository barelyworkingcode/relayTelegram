# relayTelegram

Telegram bot that bridges messages to [relayLLM](../relayLLM/) sessions. Managed as a background service by [Relay](../relay/).

## Creating the Bot

1. Open Telegram, message [@BotFather](https://t.me/BotFather), send `/newbot`
2. Choose a name and username (username must end in `bot`)
3. Copy the API token BotFather gives you
4. Get your numeric user ID by messaging [@userinfobot](https://t.me/userinfobot)
5. Build and register with Relay:
   ```bash
   ./build.sh
   ```
6. Open Relay settings and set environment variables:
   - `TELEGRAM_BOT_TOKEN` - Token from step 3
   - `TELEGRAM_ALLOWED_USER_ID` - Your numeric user ID from step 4
   - `RELAY_LLM_URL` - relayLLM server URL (default: `http://localhost:3001`)

Only the user ID in `TELEGRAM_ALLOWED_USER_ID` can interact with the bot. All other users are silently ignored.

## Linking a Chat to a Project

Before the bot will relay messages, you need to link the chat to a project:

```
/projects          # see what's available
/link myproject    # link this chat (fuzzy matches on name)
```

Once linked, every message you send is forwarded to relayLLM and the response comes back as a reply. The link persists across restarts.

```
/status            # show which project is linked and active session count
/unlink            # disconnect from the project
```

## Using Forum Topics as Separate Sessions

Telegram groups with **Topics** enabled (Settings > Topics) work as multi-session workspaces. Each topic gets its own independent session, so you can run parallel conversations against the same project.

1. Create a group and enable Topics
2. Add your bot to the group and make it admin (required for topic access)
3. `/link myproject` in any topic to link the entire group
4. Create topics for different tasks -- each one gets its own session automatically on first message

The session is tied to the topic, not the group. `/clear` in a topic resets only that topic's session.

In a regular chat (no topics), all messages share a single session.

## Commands

| Command | Description |
|---------|-------------|
| `/start` | Health check (relayLLM connectivity) |
| `/help` | Show commands (bot + session commands) |
| `/link <name>` | Link chat to a project (fuzzy match) |
| `/unlink` | Remove project link |
| `/projects` | List available projects |
| `/status` | Show linked project and session count |
| `/clear` | Reset session in current thread |

Any other `/command` (e.g. `/compact`, `/model`, `/cost`) is forwarded to relayLLM as a session command.

## Ecosystem

relayTelegram is part of the Relay ecosystem. It requires [relayLLM](https://github.com/barelyworkingcode/relayLLM) for LLM sessions.

- **[Relay](https://github.com/barelyworkingcode/relay)** -- MCP orchestrator. Manages relayTelegram as a background service.
- **[relayLLM](https://github.com/barelyworkingcode/relayLLM)** -- LLM engine. Provides the sessions and message API.

## Attribution

Telegram Bot API client: [telebot](https://gopkg.in/telebot.v4) by [tucnak](https://github.com/tucnak/telebot), MIT License.
