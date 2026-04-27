package bot_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/DavidArthurCole/EggLedgerSyncServer/bot"
)

func TestStartEmptyToken(t *testing.T) {
	closer, err := bot.Start(bot.Config{})
	if err != nil {
		t.Fatalf("expected nil error for empty token, got: %v", err)
	}
	if closer == nil {
		t.Fatal("expected non-nil closer, got nil")
	}
	closer() // must not panic
}

func TestVerifyResponse(t *testing.T) {
	cfg := bot.Config{
		BuildSHA256:  "abc123def456",
		BuildVersion: "v1.0.0",
		BuildDate:    "2026-04-27T00:00:00Z",
	}
	resp := bot.VerifyResponse(cfg)

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Data == nil {
		t.Fatal("expected non-nil response data")
	}
	if len(resp.Data.Embeds) == 0 {
		t.Fatal("expected at least one embed")
	}

	embed := resp.Data.Embeds[0]
	fieldValues := make(map[string]string)
	for _, f := range embed.Fields {
		fieldValues[f.Name] = f.Value
	}

	if got := fieldValues["SHA256"]; got != cfg.BuildSHA256 {
		t.Errorf("SHA256: want %q, got %q", cfg.BuildSHA256, got)
	}
	if got := fieldValues["Version"]; got != cfg.BuildVersion {
		t.Errorf("Version: want %q, got %q", cfg.BuildVersion, got)
	}
	if got := fieldValues["Built"]; got != cfg.BuildDate {
		t.Errorf("Built: want %q, got %q", cfg.BuildDate, got)
	}
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Errorf("expected ephemeral flag, got %d", resp.Data.Flags)
	}
}
