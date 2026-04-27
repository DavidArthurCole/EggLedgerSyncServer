package bot_test

import (
	"strings"
	"testing"

	"github.com/DavidArthurCole/EggLedgerSyncServer/bot"
	"github.com/bwmarrin/discordgo"
)

func TestUpdateResponse_Success(t *testing.T) {
	resp := bot.UpdateResponse(true, "")
	if resp == nil || resp.Data == nil {
		t.Fatal("nil response or data")
	}
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Errorf("expected ephemeral flag, got %d", resp.Data.Flags)
	}
	if len(resp.Data.Embeds) == 0 {
		t.Fatal("expected at least one embed")
	}
	embed := resp.Data.Embeds[0]
	if !strings.Contains(embed.Title, "Updated") {
		t.Errorf("success title should contain 'Updated', got %q", embed.Title)
	}
	if embed.Description != "" {
		t.Errorf("success description should be empty, got %q", embed.Description)
	}
}

func TestUpdateResponse_Failure(t *testing.T) {
	tail := "Error: build failed\nstep 5/7 FAILED"
	resp := bot.UpdateResponse(false, tail)
	if resp == nil || resp.Data == nil {
		t.Fatal("nil response or data")
	}
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Errorf("expected ephemeral flag, got %d", resp.Data.Flags)
	}
	if len(resp.Data.Embeds) == 0 {
		t.Fatal("expected at least one embed")
	}
	embed := resp.Data.Embeds[0]
	if !strings.Contains(strings.ToLower(embed.Title), "failed") {
		t.Errorf("failure title should contain 'failed', got %q", embed.Title)
	}
	if !strings.Contains(embed.Description, tail) {
		t.Errorf("description should contain tail output\ngot:  %q\nwant: contains %q", embed.Description, tail)
	}
	if !strings.Contains(embed.Description, "```") {
		t.Error("description should wrap tail in a code block")
	}
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name   string
		member *discordgo.Member
		want   bool
	}{
		{"nil member", nil, false},
		{"no permissions", &discordgo.Member{Permissions: 0}, false},
		{"administrator bit set", &discordgo.Member{Permissions: discordgo.PermissionAdministrator}, true},
		{"admin plus other perms", &discordgo.Member{Permissions: discordgo.PermissionAdministrator | discordgo.PermissionManageMessages}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{Member: tt.member},
			}
			if got := bot.IsAdmin(i); got != tt.want {
				t.Errorf("IsAdmin = %v, want %v", got, tt.want)
			}
		})
	}
}
