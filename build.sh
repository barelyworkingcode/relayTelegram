#!/bin/bash
set -e
cd "$(dirname "$0")"

go build -o relayTelegram .
echo "Built relayTelegram binary."

# Only register if not already registered (avoid overwriting env vars set in Relay UI)
if /Applications/Relay.app/Contents/MacOS/relay service list 2>/dev/null | grep -q "Relay Telegram"; then
  echo "Already registered with Relay. Binary updated in place."
else
  /Applications/Relay.app/Contents/MacOS/relay service register \
    --name "Relay Telegram" \
    --command "$(pwd)/relayTelegram" \
    --autostart
  echo ""
  echo "Registered with Relay. Open Relay settings and set these environment variables:"
  echo "  TELEGRAM_BOT_TOKEN=<your bot token from BotFather>"
  echo "  TELEGRAM_ALLOWED_USER_ID=<your numeric Telegram user ID>"
  echo "  EVE_URL=http://localhost:3000"
fi
