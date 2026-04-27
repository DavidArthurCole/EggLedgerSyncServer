package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

// IsAdmin reports whether the interaction came from a member with the Administrator permission.
func IsAdmin(i *discordgo.InteractionCreate) bool {
	return i.Member != nil && i.Member.Permissions&discordgo.PermissionAdministrator != 0
}

// AlreadyUpToDateEmbed builds the public embed when the server is already on the latest commit.
func AlreadyUpToDateEmbed(hash string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Already up to date.",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Current", Value: fmt.Sprintf("[`%s`](%s)", hash, commitURL(hash)), Inline: true},
		},
	}
}

// SuccessEmbed builds the public green embed for a successful /updateserver.
func SuccessEmbed(fromHash, toHash string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Updated",
		Color: 0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "From", Value: fmt.Sprintf("[`%s`](%s)", fromHash, commitURL(fromHash)), Inline: true},
			{Name: "To", Value: fmt.Sprintf("[`%s`](%s)", toHash, commitURL(toHash)), Inline: true},
		},
	}
}

// FailureEmbed builds the ephemeral red embed for a failed /updateserver.
func FailureEmbed(tail string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Update failed.",
		Description: fmt.Sprintf("```\n%s\n```", tail),
		Color:       0xED4245,
	}
}

type deployAgentResponse struct {
	OK              bool   `json:"ok"`
	AlreadyUpToDate bool   `json:"alreadyUpToDate"`
	Tail            string `json:"tail"`
	FromHash        string `json:"fromHash"`
	ToHash          string `json:"toHash"`
}

func callDeployAgent(agentURL, secret string) (ok, alreadyUpToDate bool, tail, fromHash, toHash string, err error) {
	client := &http.Client{Timeout: 120 * time.Second}
	req, err := http.NewRequest(http.MethodPost, agentURL, nil)
	if err != nil {
		return false, false, "", "", "", errors.Wrap(err, "callDeployAgent: new request")
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := client.Do(req)
	if err != nil {
		return false, false, "", "", "", errors.Wrap(err, "callDeployAgent: do request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, false, fmt.Sprintf("deploy agent returned %s", resp.Status), "", "", nil
	}
	var result deployAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, false, "could not decode deploy agent response", "", "", nil
	}
	return result.OK, result.AlreadyUpToDate, result.Tail, result.FromHash, result.ToHash, nil
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
	// Defer without ephemeral — success result is visible to the channel.
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("bot: updateserver: deferred respond: %v", err)
		return
	}
	go func() {
		ok, alreadyUpToDate, tail, fromHash, toHash, err := callDeployAgent(cfg.DeployAgentURL, cfg.DeployAgentSecret)
		if err != nil {
			ok = false
			tail = err.Error()
		}
		var embed *discordgo.MessageEmbed
		switch {
		case alreadyUpToDate:
			embed = AlreadyUpToDateEmbed(fromHash)
		case ok:
			embed = SuccessEmbed(fromHash, toHash)
		}
		if embed != nil {
			embeds := []*discordgo.MessageEmbed{embed}
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &embeds,
			}); err != nil {
				log.Printf("bot: updateserver: edit response: %v", err)
			}
		} else {
			if err := s.InteractionResponseDelete(i.Interaction); err != nil {
				log.Printf("bot: updateserver: delete response: %v", err)
			}
			if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:  discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{FailureEmbed(tail)},
			}); err != nil {
				log.Printf("bot: updateserver: followup: %v", err)
			}
		}
	}()
}
