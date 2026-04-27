package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

// Config holds all configuration needed to start the bot.
type Config struct {
	Token        string
	AppID        string
	GuildID      string
	BuildSHA256  string
	BuildVersion string
	BuildDate    string
}

// Start opens a Discord Gateway session, sets bot presence, and registers slash commands.
// Always returns a non-nil closer. Returns (noop, nil) if Token is empty.
// Returns (noop, error) if the session fails to open.
func Start(cfg Config) (func(), error) {
	noop := func() { /* intentional no-op: allows caller to unconditionally defer without a nil check */ }

	if cfg.Token == "" {
		log.Println("bot: DISCORD_BOT_TOKEN not set, skipping bot")
		return noop, nil
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return noop, errors.Wrap(err, "bot.Start: new session")
	}

	session.AddHandler(handleInteraction(cfg))

	if err := session.Open(); err != nil {
		return noop, errors.Wrap(err, "bot.Start: open gateway")
	}

	if err := session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{
			{Name: "EggLedger", Type: discordgo.ActivityTypeGame},
		},
	}); err != nil {
		log.Printf("bot: UpdateStatusComplex: %v", err)
	}

	var registeredCommands []*discordgo.ApplicationCommand
	if cfg.AppID != "" && cfg.GuildID != "" {
		registeredCommands = registerCommands(session, cfg)
	} else {
		log.Println("bot: AppID or GuildID not set, skipping command registration")
	}

	return func() {
		unregisterCommands(session, cfg, registeredCommands)
		if err := session.Close(); err != nil {
			log.Printf("bot: session.Close: %v", err)
		}
	}, nil
}
