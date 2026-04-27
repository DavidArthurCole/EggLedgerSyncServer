package bot

import "github.com/bwmarrin/discordgo"

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
						{Name: "SHA256", Value: cfg.BuildSHA256},
						{Name: "Version", Value: cfg.BuildVersion, Inline: true},
						{Name: "Built", Value: cfg.BuildDate, Inline: true},
					},
				},
			},
		},
	}
}
