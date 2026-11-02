package token

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"testing"
	"time"
)

func TestTokenExpiryVariantsAndInvalidation(t *testing.T) {
	for _, expiry := range []string{`"7200"`, `7200`} {
		var response qqBotTokenRsp
		if err := json.Unmarshal([]byte(`{"access_token":"fake","expires_in":`+expiry+`}`), &response); err != nil || response.ExpiresIn != 7200 {
			t.Fatal("expiry protocol variant failed")
		}
	}
	source := NewQQBotTokenSource(&QQBotCredentials{AppID: "test", AppSecret: "fake"}).(*QQBotTokenSource)
	source.cachedToken.Store(&oauth2.Token{AccessToken: "fake", Expiry: time.Now().Add(time.Hour)})
	source.Invalidate()
	if source.cachedToken.Load().(*oauth2.Token).Valid() {
		t.Fatal("invalidated token still valid")
	}
}
