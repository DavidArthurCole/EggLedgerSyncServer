package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

// IsAdmin reports whether the interaction came from a member with the Administrator permission.
func IsAdmin(i *discordgo.InteractionCreate) bool {
	return i.Member != nil && i.Member.Permissions&discordgo.PermissionAdministrator != 0
}

// UpdateResponse builds the ephemeral embed InteractionResponse for /updateserver.
// On success, tail is ignored. On failure, tail appears in a code block.
func UpdateResponse(success bool, tail string) *discordgo.InteractionResponse {
	title := "Updated ✓"
	var description string
	if !success {
		title = "Update failed."
		description = fmt.Sprintf("```\n%s\n```", tail)
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       0x5865F2,
				},
			},
		},
	}
}

type deployAgentResponse struct {
	OK   bool   `json:"ok"`
	Tail string `json:"tail"`
}

func callDeployAgent(agentURL, secret string) (ok bool, tail string, err error) {
	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest(http.MethodPost, agentURL, nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error(), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("deploy agent returned %s", resp.Status), nil
	}
	var result deployAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "could not decode deploy agent response", nil
	}
	return result.OK, result.Tail, nil
}

func handleUpdateServer(s *discordgo.Session, i *discordgo.InteractionCreate, cfg Config) {
	if !IsAdmin(i) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Not authorised.",
			},
		}); err != nil {
			log.Printf("bot: updateserver: respond not-authorised: %v", err)
		}
		return
	}
	if cfg.DeployAgentURL == "" || cfg.DeployAgentSecret == "" {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Deploy agent not configured.",
			},
		}); err != nil {
			log.Printf("bot: updateserver: respond not-configured: %v", err)
		}
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("bot: updateserver: deferred respond: %v", err)
		return
	}
	go func() {
		ok, tail, err := callDeployAgent(cfg.DeployAgentURL, cfg.DeployAgentSecret)
		if err != nil {
			ok = false
			tail = err.Error()
		}
		resp := UpdateResponse(ok, tail)
		embeds := resp.Data.Embeds
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embeds,
		}); err != nil {
			log.Printf("bot: updateserver: edit response: %v", err)
		}
	}()
}
