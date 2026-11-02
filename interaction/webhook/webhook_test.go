package webhook

import (
	"encoding/json"
	"github.com/kylin930/botgo-plus/dto"
	"github.com/kylin930/botgo-plus/event"
	"github.com/kylin930/botgo-plus/interaction/signature"
	"github.com/kylin930/botgo-plus/token"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenHeartbeatACK(t *testing.T) {
	j := GenHeartbeatACK(1314)
	if !strings.Contains(j, "11") || !strings.Contains(j, "1314") {
		t.Error("GenHeartbeatACK error")
	}
}

func TestGenDispatchACK(t *testing.T) {
	j := GenDispatchACK(false)
	if !strings.Contains(j, "12") || !strings.Contains(j, "1") {
		t.Error("GenDispatchACK error")
	}
	j = GenDispatchACK(true)
	if !strings.Contains(j, "12") || !strings.Contains(j, "0") {
		t.Error("GenDispatchACK error")
	}
}

func TestSignedRawCallbacksAndValidation(t *testing.T) {
	credentials := &token.QQBotCredentials{AppID: "local-app", AppSecret: "local-secret"}
	old := event.DefaultHandlers
	defer func() { event.DefaultHandlers = old }()
	calls := 0
	event.DefaultHandlers.Plain = func(p *dto.WSPayload, raw []byte) error { calls++; return nil }
	request := func(body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/callback", strings.NewReader(body))
		r.ContentLength = -1
		r.Header.Set(signature.HeaderTimestamp, "12345")
		sig, err := signature.Generate(credentials.AppSecret, r.Header, []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set(signature.HeaderSig, sig)
		if mutate != nil {
			mutate(r)
		}
		w := httptest.NewRecorder()
		HTTPHandler(w, r, credentials)
		return w
	}
	dispatch := `{"op":0,"id":"event","t":"LOCAL_CALLBACK","d":{"content":"private-message"}}`
	w := request(dispatch, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"d":0`) || calls != 1 {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
	for _, tc := range []struct {
		name, body string
		mutate     func(*http.Request)
		status     int
	}{
		{"unsigned", dispatch, func(r *http.Request) { r.Header.Del(signature.HeaderSig) }, 401},
		{"missing-timestamp", dispatch, func(r *http.Request) { r.Header.Del(signature.HeaderTimestamp) }, 401},
		{"tampered", dispatch, func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader(dispatch + " ")) }, 401},
		{"malformed", "{", nil, 400}, {"invalid-heartbeat", `{"op":1,"d":"oops"}`, nil, 400},
		{"invalid-challenge", `{"op":13,"d":{"plain_token":"x"}}`, nil, 400},
		{"oversized", strings.Repeat("x", (2<<20)+1), nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := request(tc.body, tc.mutate)
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	if calls != 1 {
		t.Fatal("invalid request reached event handlers")
	}
	challenge := `{"op":13,"d":{"plain_token":"challenge","event_ts":"67890"}}`
	w = request(challenge, nil)
	var response dto.WHValidationRsp
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	header := http.Header{}
	header.Set(signature.HeaderTimestamp, "67890")
	header.Set(signature.HeaderSig, response.Signature)
	pass, err := signature.Verify(credentials.AppSecret, header, []byte(response.PlainToken))
	if w.Code != 200 || response.PlainToken != "challenge" || err != nil || !pass {
		t.Fatal("invalid validation signature")
	}
	if result := GenValidationACK(nil, http.Header{}, credentials.AppSecret); result != nil {
		t.Fatal("nil validation accepted")
	}
	w = request(`{"op":1,"d":42}`, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"d":42`) {
		t.Fatal(w.Body.String())
	}
}
