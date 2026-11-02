package token

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenFetchCacheInvalidationAndRefresh(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req qqBotTokenReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.AppID != "app" || req.ClientSecret != "fake-secret" {
			t.Error("credentials request mismatch")
		}
		n := atomic.AddInt32(&count, 1)
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"fake-%d","expires_in":7200}`, n)
	}))
	defer server.Close()
	source := NewQQBotTokenSource(&QQBotCredentials{AppID: "app", AppSecret: "fake-secret"}).(*QQBotTokenSource)
	source.endpoint = server.URL
	source.client = server.Client()
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := source.Token(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&count) != 1 {
		t.Fatal("concurrent token fetch not coalesced")
	}
	source.Invalidate()
	tk, err := source.Token()
	if err != nil || tk.AccessToken != "fake-2" || atomic.LoadInt32(&count) != 2 {
		t.Fatal("token did not refresh")
	}
}
func TestTokenRejectsInvalidOrFailedResponses(t *testing.T) {
	for _, body := range []string{`{"access_token":"fake","expires_in":0}`, `{"expires_in":"7200"}`, `{"code":400,"message":"fake-secret","expires_in":"7200"}`, `{"access_token":"fake-secret","expires_in":"fake-secret"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
		source := NewQQBotTokenSource(&QQBotCredentials{AppID: "app", AppSecret: "fake-secret"}).(*QQBotTokenSource)
		source.endpoint = server.URL
		source.client = server.Client()
		_, err := source.Token()
		server.Close()
		if err == nil || strings.Contains(err.Error(), "fake-secret") {
			t.Fatal("invalid token accepted or leaked")
		}
	}
}
