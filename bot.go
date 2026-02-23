package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tele "gopkg.in/telebot.v4"
)

type Bot struct {
	bot         *tele.Bot
	cfg         Config
	eve         *EveClient
	mappings    *Mappings
	botCommands map[string]bool
}

func NewBot(cfg Config, eve *EveClient, mappings *Mappings) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 30 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	bot := &Bot{
		bot:      b,
		cfg:      cfg,
		eve:      eve,
		mappings: mappings,
	}

	bot.registerHandlers()
	return bot, nil
}

func (b *Bot) Start() {
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.bot.Stop()
}

func (b *Bot) isAllowed(c tele.Context) bool {
	return c.Sender().ID == b.cfg.AllowedUserID
}

func (b *Bot) registerHandlers() {
	// Bot-owned commands. Everything else starting with / gets forwarded to Eve.
	b.botCommands = map[string]bool{
		"/start":    true,
		"/link":     true,
		"/unlink":   true,
		"/projects": true,
		"/status":   true,
		"/clear":    true,
		"/help":     true,
	}

	b.bot.Handle("/start", b.handleStartRaw)
	b.bot.Handle("/help", b.authGuard(b.handleHelp))
	b.bot.Handle("/link", b.authGuard(b.handleLink))
	b.bot.Handle("/unlink", b.authGuard(b.handleUnlink))
	b.bot.Handle("/projects", b.authGuard(b.handleProjects))
	b.bot.Handle("/status", b.authGuard(b.handleStatus))
	b.bot.Handle("/clear", b.authGuard(b.handleClear))
	b.bot.Handle(tele.OnText, b.authGuard(b.handleMessage))

	// Middleware: intercept /commands not in botCommands and forward to Eve via handleMessage.
	// Runs before registered handlers, so unrecognized commands (e.g. /compact, /model) never
	// reach telebot's "not found" path. Plain text and known commands fall through to next().
	b.bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			msg := c.Message()
			if msg != nil && msg.Text != "" && strings.HasPrefix(msg.Text, "/") {
				cmd := strings.SplitN(msg.Text, " ", 2)[0]
				cmd = strings.SplitN(cmd, "@", 2)[0] // strip @botname
				if !b.botCommands[cmd] {
					if !b.isAllowed(c) {
						return nil
					}
					return b.handleMessage(c)
				}
			}
			return next(c)
		}
	})
}

func (b *Bot) authGuard(handler tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if !b.isAllowed(c) {
			return nil // Silent ignore
		}
		return handler(c)
	}
}

func chatID(c tele.Context) string {
	return strconv.FormatInt(c.Chat().ID, 10)
}

func threadID(c tele.Context) string {
	if c.Message().ThreadID != 0 {
		return strconv.Itoa(c.Message().ThreadID)
	}
	return "default"
}

func (b *Bot) handleStartRaw(c tele.Context) error {
	if !b.isAllowed(c) {
		return c.Reply("New phone, who dis?")
	}
	_, err := b.eve.ListProjects()
	if err != nil {
		log.Printf("Eve health check failed: %v", err)
		return c.Reply("Online, but Eve is unreachable.")
	}
	return c.Reply("Online. Eve is connected.")
}

func (b *Bot) handleHelp(c tele.Context) error {
	err := c.Reply(`Bot commands:
/start - Health check
/help - This message
/link <name> - Link chat to an Eve project
/unlink - Remove link
/projects - List Eve projects
/status - Show current mapping
/clear - Start new session

Other /commands are forwarded to Eve.`)
	if err != nil {
		return err
	}

	// Forward /help to Eve if there's an active session
	cid := chatID(c)
	tid := threadID(c)
	sm := b.mappings.GetSession(cid, tid)
	if sm != nil {
		result, err := b.eve.SendMessage(sm.EveSessionID, "/help")
		if err == nil && result.Response != "" {
			return c.Reply(result.Response)
		}
	}

	return nil
}

func (b *Bot) handleLink(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Reply("Usage: /link <projectName>")
	}

	query := strings.Join(args, " ")

	projects, err := b.eve.ListProjects()
	if err != nil {
		return c.Reply(fmt.Sprintf("Failed to reach Eve: %v", err))
	}

	// Fuzzy match: case-insensitive substring
	var match *EveProject
	var matches []EveProject
	queryLower := strings.ToLower(query)
	for i, p := range projects {
		if p.Disabled {
			continue
		}
		if strings.ToLower(p.Name) == queryLower {
			match = &projects[i]
			break
		}
		if strings.Contains(strings.ToLower(p.Name), queryLower) {
			matches = append(matches, p)
		}
	}

	if match == nil && len(matches) == 1 {
		match = &matches[0]
	}
	if match == nil && len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = fmt.Sprintf("  %s", m.Name)
		}
		return c.Reply(fmt.Sprintf("Multiple matches:\n%s\n\nBe more specific.", strings.Join(names, "\n")))
	}
	if match == nil {
		return c.Reply(fmt.Sprintf("No project found matching %q", query))
	}

	if err := b.mappings.LinkChat(chatID(c), match.ID, match.Name); err != nil {
		return c.Reply(fmt.Sprintf("Failed to save mapping: %v", err))
	}

	return c.Reply(fmt.Sprintf("Linked to project: %s", match.Name))
}

func (b *Bot) handleUnlink(c tele.Context) error {
	cid := chatID(c)
	cm := b.mappings.GetChatMapping(cid)
	if cm == nil {
		return c.Reply("This chat is not linked to any project.")
	}

	if err := b.mappings.UnlinkChat(cid); err != nil {
		return c.Reply(fmt.Sprintf("Failed to unlink: %v", err))
	}

	return c.Reply(fmt.Sprintf("Unlinked from project: %s", cm.ProjectName))
}

func (b *Bot) handleProjects(c tele.Context) error {
	projects, err := b.eve.ListProjects()
	if err != nil {
		return c.Reply(fmt.Sprintf("Failed to reach Eve: %v", err))
	}

	if len(projects) == 0 {
		return c.Reply("No projects found.")
	}

	var lines []string
	for _, p := range projects {
		status := ""
		if p.Disabled {
			status = " (disabled)"
		}
		lines = append(lines, fmt.Sprintf("  %s [%s]%s", p.Name, p.Model, status))
	}

	return c.Reply(fmt.Sprintf("Projects:\n%s", strings.Join(lines, "\n")))
}

func (b *Bot) handleStatus(c tele.Context) error {
	cid := chatID(c)
	cm := b.mappings.GetChatMapping(cid)
	if cm == nil {
		return c.Reply("This chat is not linked. Use /link <projectName>")
	}

	sessionCount := len(cm.Sessions)
	return c.Reply(fmt.Sprintf("Project: %s\nSessions: %d", cm.ProjectName, sessionCount))
}

func (b *Bot) handleClear(c tele.Context) error {
	cid := chatID(c)
	tid := threadID(c)

	if err := b.mappings.ClearSession(cid, tid); err != nil {
		return c.Reply(fmt.Sprintf("Failed to clear session: %v", err))
	}

	return c.Reply("Session cleared. Next message will start a new conversation.")
}

func (b *Bot) handleMessage(c tele.Context) error {
	cid := chatID(c)
	tid := threadID(c)
	text := c.Text()

	cm := b.mappings.GetChatMapping(cid)
	if cm == nil {
		return c.Reply("This chat is not linked to a project. Use /link <projectName>")
	}

	// Get or create session
	sm := b.mappings.GetSession(cid, tid)
	if sm == nil {
		sessionName := ""
		if tid != "default" {
			sessionName = fmt.Sprintf("Telegram thread %s", tid)
		}

		result, err := b.eve.CreateSession(cm.ProjectID, sessionName)
		if err != nil {
			return c.Reply(fmt.Sprintf("Failed to create session: %v", err))
		}

		if err := b.mappings.SetSession(cid, tid, result.SessionID); err != nil {
			log.Printf("Failed to save session mapping: %v", err)
		}

		sm = &SessionMapping{EveSessionID: result.SessionID}
	}

	// Send typing indicator, repeat every 5s
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		_ = c.Notify(tele.Typing)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = c.Notify(tele.Typing)
			}
		}
	}()

	// Send message to Eve
	result, err := b.eve.SendMessage(sm.EveSessionID, text)
	close(done)

	if err != nil {
		// If session not found, clear mapping so next message creates a new one
		if strings.Contains(err.Error(), "not found") {
			_ = b.mappings.ClearSession(cid, tid)
			return c.Reply("Session expired. Send your message again to start a new conversation.")
		}
		return c.Reply(fmt.Sprintf("Error: %v", err))
	}

	// Update last active
	_ = b.mappings.SetSession(cid, tid, sm.EveSessionID)

	// Send response, splitting at Telegram's 4096 char limit
	return b.sendLongMessage(c, result.Response)
}

func (b *Bot) sendLongMessage(c tele.Context, text string) error {
	const maxLen = 4096 // Telegram's limit is in UTF-16 code units, but rune count is a safe approximation

	if utf8.RuneCountInString(text) <= maxLen {
		return c.Reply(text)
	}

	// Split at paragraph boundaries using rune-aware indexing
	for len(text) > 0 {
		if utf8.RuneCountInString(text) <= maxLen {
			return c.Reply(text)
		}

		// Find the byte index of the maxLen-th rune
		byteLimit := runeByteIndex(text, maxLen)

		// Find last double newline before limit
		cut := strings.LastIndex(text[:byteLimit], "\n\n")
		if cut < byteLimit/2 {
			// Fall back to single newline
			cut = strings.LastIndex(text[:byteLimit], "\n")
		}
		if cut < byteLimit/2 {
			// Hard split at rune boundary
			cut = byteLimit
		}

		chunk := text[:cut]
		text = strings.TrimLeft(text[cut:], "\n")

		if err := c.Reply(chunk); err != nil {
			return err
		}

		// Small delay between chunks to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// runeByteIndex returns the byte index of the n-th rune in s.
// If s has fewer than n runes, returns len(s).
func runeByteIndex(s string, n int) int {
	i := 0
	for count := 0; count < n && i < len(s); count++ {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return i
}
