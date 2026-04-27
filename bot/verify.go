package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const githubRepo = "https://github.com/DavidArthurCole/EggLedgerSyncServer"

func commitURL(version string) string {
	return fmt.Sprintf("%s/commit/%s", githubRepo, version)
}

// VerifyResponse builds an ephemeral embed InteractionResponse containing the
// server's build identity fields from cfg.
func VerifyResponse(cfg Config) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "EggLedger Sync Server",
					Color: 0x5865F2,
					Fields: []*discordgo.MessageEmbedField{
						{Name: "SHA256", Value: fmt.Sprintf("[%s](%s)", cfg.BuildSHA256, commitURL(cfg.BuildVersion))},
						{Name: "Version", Value: fmt.Sprintf("[%s](%s)", cfg.BuildVersion, commitURL(cfg.BuildVersion)), Inline: true},
						{Name: "Built", Value: cfg.BuildDate, Inline: true},
					},
				},
			},
		},
	}
}
