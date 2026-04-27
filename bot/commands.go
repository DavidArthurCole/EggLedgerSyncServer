package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var commandDefinitions = []*discordgo.ApplicationCommand{
	{
		Name:        "verify",
		Description: "Show the running server's build identity (SHA256, version, build date).",
	},
}

func registerCommands(s *discordgo.Session, cfg Config) []*discordgo.ApplicationCommand {
	registered := make([]*discordgo.ApplicationCommand, 0, len(commandDefinitions))
	for _, cmd := range commandDefinitions {
		created, err := s.ApplicationCommandCreate(cfg.AppID, cfg.GuildID, cmd)
		if err != nil {
			log.Printf("bot: register command %q: %v", cmd.Name, err)
			continue
		}
		registered = append(registered, created)
	}
	return registered
}

func unregisterCommands(s *discordgo.Session, cfg Config, commands []*discordgo.ApplicationCommand) {
	for _, cmd := range commands {
		if err := s.ApplicationCommandDelete(cfg.AppID, cfg.GuildID, cmd.ID); err != nil {
			log.Printf("bot: delete command %q: %v", cmd.Name, err)
		}
	}
}

func handleInteraction(cfg Config) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		data := i.ApplicationCommandData()
		switch data.Name {
		case "verify":
			if err := s.InteractionRespond(i.Interaction, VerifyResponse(cfg)); err != nil {
				log.Printf("bot: verify respond: %v", err)
			}
		}
	}
}
