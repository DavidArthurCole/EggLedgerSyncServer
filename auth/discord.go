package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

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

// HandleCallback exchanges the OAuth2 code for a Discord user ID,
// then calls storePendingFn with the state, a new session token, and the Discord user ID.
func HandleCallback(ctx context.Context, code, state string, storePendingFn func(state, token, discordID string) error) error {
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
	var user struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return errors.Wrap(err, "HandleCallback: decode user")
	}
	sessionToken := randomHex(32)
	return storePendingFn(state, sessionToken, user.ID)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
