package bot_test

import (
	"strings"
	"testing"

	"github.com/DavidArthurCole/EggLedgerSyncServer/bot"
	"github.com/bwmarrin/discordgo"
)

func TestAlreadyUpToDateEmbed(t *testing.T) {
	embed := bot.AlreadyUpToDateEmbed("abc1234")
	if embed == nil {
		t.Fatal("nil embed")
	}
	if embed.Color != 0x5865F2 {
		t.Errorf("expected blurple color 0x5865F2, got 0x%X", embed.Color)
	}
	if !strings.Contains(embed.Title, "up to date") {
		t.Errorf("title should contain 'up to date', got %q", embed.Title)
	}
	if len(embed.Fields) == 0 {
		t.Fatal("expected at least one field")
	}
	if !strings.Contains(embed.Fields[0].Value, "abc1234") {
		t.Errorf("field should contain hash, got %q", embed.Fields[0].Value)
	}
	if !strings.Contains(embed.Fields[0].Value, "https://github.com") {
		t.Error("field should be a link")
	}
}

func TestSuccessEmbed(t *testing.T) {
	embed := bot.SuccessEmbed("abc1234", "def5678")
	if embed == nil {
		t.Fatal("nil embed")
	}
	if embed.Color != 0x57F287 {
		t.Errorf("expected green color 0x57F287, got 0x%X", embed.Color)
	}
	if !strings.Contains(embed.Title, "Updated") {
		t.Errorf("title should contain 'Updated', got %q", embed.Title)
	}
	fieldValues := make(map[string]string)
	for _, f := range embed.Fields {
		fieldValues[f.Name] = f.Value
	}
	if !strings.Contains(fieldValues["From"], "abc1234") {
		t.Errorf("From field should contain fromHash, got %q", fieldValues["From"])
	}
	if !strings.Contains(fieldValues["To"], "def5678") {
		t.Errorf("To field should contain toHash, got %q", fieldValues["To"])
	}
	if !strings.Contains(fieldValues["From"], "https://github.com") {
		t.Error("From field should be a link")
	}
}

func TestFailureEmbed(t *testing.T) {
	tail := "Error: build failed\nstep 5/7 FAILED"
	embed := bot.FailureEmbed(tail)
	if embed == nil {
		t.Fatal("nil embed")
	}
	if embed.Color != 0xED4245 {
		t.Errorf("expected red color 0xED4245, got 0x%X", embed.Color)
	}
	if !strings.Contains(strings.ToLower(embed.Title), "failed") {
		t.Errorf("title should contain 'failed', got %q", embed.Title)
	}
	if !strings.Contains(embed.Description, tail) {
		t.Errorf("description should contain tail, got %q", embed.Description)
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
