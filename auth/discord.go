package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

var _oauthConfig *oauth2.Config

func Init(clientID, clientSecret, redirectURL string) {
	_oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"identify"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
}

func AuthURL() (url, state string) {
	state = randomHex(16)
	url = _oauthConfig.AuthCodeURL(state)
	return url, state
}

// DiscordUser holds the fields we capture from the Discord /users/@me endpoint.
type DiscordUser struct {
	ID       string
	Username string
	AvatarURL string
}

// HandleCallback exchanges the OAuth2 code for Discord user info,
// then calls storePendingFn with the state, a new session token, and the Discord user.
func HandleCallback(ctx context.Context, code, state string, storePendingFn func(state, token string, user DiscordUser) error) error {
	token, err := _oauthConfig.Exchange(ctx, code)
	if err != nil {
		return errors.Wrap(err, "HandleCallback: exchange")
	}
	client := _oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return errors.Wrap(err, "HandleCallback: get user")
	}
	defer resp.Body.Close()
	var raw struct {
		ID     string `json:"id"`
		Username string `json:"username"`
		Avatar string `json:"avatar"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return errors.Wrap(err, "HandleCallback: decode user")
	}
	avatarURL := ""
	if raw.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", raw.ID, raw.Avatar)
	}
	sessionToken := randomHex(32)
	return storePendingFn(state, sessionToken, DiscordUser{ID: raw.ID, Username: raw.Username, AvatarURL: avatarURL})
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
