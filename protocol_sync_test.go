package botgo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	botgo "github.com/kylin930/botgo-plus"
	"github.com/kylin930/botgo-plus/dto"
	"github.com/kylin930/botgo-plus/dto/keyboard"
	"github.com/kylin930/botgo-plus/errs"
	sdklog "github.com/kylin930/botgo-plus/log"
	"github.com/kylin930/botgo-plus/openapi"
	"golang.org/x/oauth2"
)

func localAPI(t *testing.T, server *httptest.Server, tokens oauth2.TokenSource) openapi.OpenAPI {
	t.Helper()
	if tokens == nil {
		tokens = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake-token", TokenType: "QQBot"})
	}
	api := botgo.NewOpenAPI("local-app", tokens)
	return api.(interface{ WithBaseURL(string) openapi.OpenAPI }).WithBaseURL(server.URL)
}
func TestProtocolSyncMessageAndMediaHTTP(t *testing.T) {
	for _, group := range []bool{false, true} {
		t.Run(fmt.Sprint(group), func(t *testing.T) {
			var bodies []map[string]interface{}
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("X-Union-Appid") != "local-app" || r.Header.Get("Authorization") != "QQBot fake-token" {
					t.Error("incorrect SDK method/auth")
				}
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				bodies = append(bodies, body)
				paths = append(paths, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Tps-Trace-ID", "trace-local")
				if strings.HasSuffix(r.URL.Path, "/files") {
					io.WriteString(w, `{"file_info":"opaque/not-base64!","ttl":300}`)
				} else {
					io.WriteString(w, `{"id":"sent","timestamp":"2026-07-21T10:30:00+08:00"}`)
				}
			}))
			defer server.Close()
			api := localAPI(t, server, nil)
			post := func(message dto.APIMessage) (*dto.Message, error) {
				if group {
					return api.PostGroupMessage(context.Background(), "target", message)
				}
				return api.PostC2CMessage(context.Background(), "target", message)
			}
			for _, kind := range []dto.MessageType{dto.TextMsg, dto.MarkdownMsg} {
				m := &dto.MessageToCreate{MsgType: kind, MsgID: "incoming", MsgSeq: 1, Content: "hello"}
				if kind == dto.MarkdownMsg {
					m.Content = ""
					m.Markdown = &dto.Markdown{Content: "**hello**"}
					m.Keyboard = &keyboard.MessageKeyboard{ID: "template"}
				}
				result, err := post(m)
				if err != nil {
					t.Fatal(err)
				}
				if result.ID != "sent" || result.TraceID != "trace-local" {
					t.Fatal("request trace or ID lost")
				}
			}
			// Each independent /files request has no passive association/sequence.
			for _, upload := range []*dto.RichMediaMessage{{FileType: 1, URL: "https://example.invalid/image", SrvSendMsg: false}, {FileType: 2, URL: "https://example.invalid/video", SrvSendMsg: false}, {FileType: 1, FileData: "aW1hZ2U=", SrvSendMsg: false}} {
				result, err := post(upload)
				if err != nil {
					t.Fatal(err)
				}
				if result.FileInfoRaw != "opaque/not-base64!" {
					t.Fatal("file info changed")
				}
				_, err = post(&dto.MessageToCreate{MsgType: dto.RichMediaMsg, Content: " ", MsgID: "incoming", MsgSeq: 2, Media: &dto.MediaInfo{FileInfo: result.FileInfo, FileInfoRaw: result.FileInfoRaw}})
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := post(&dto.MessageToCreate{MsgType: dto.TextMsg, EventID: "event", MsgSeq: 3, Content: "event reply"}); err != nil {
				t.Fatal(err)
			}
			for i, body := range bodies {
				root := "/v2/users/target"
				if group {
					root = "/v2/groups/target"
				}
				if strings.HasSuffix(paths[i], "/files") {
					if paths[i] != root+"/files" || body["srv_send_msg"] != false || body["msg_seq"] != nil || body["msg_id"] != nil {
						t.Fatal("upload consumed reply association", body)
					}
					continue
				}
				if paths[i] != root+"/messages" || body["msg_type"] == nil {
					t.Fatal("wrong final endpoint/type", body)
				}
				if i == len(bodies)-1 {
					if body["event_id"] != "event" || body["msg_id"] != nil || body["msg_seq"] != float64(3) {
						t.Fatal(body)
					}
					continue
				}
				if body["msg_id"] != "incoming" || body["msg_seq"] == nil {
					t.Fatal("association lost", body)
				}
				if body["msg_type"] == float64(7) && body["media"].(map[string]interface{})["file_info"] != "opaque/not-base64!" {
					t.Fatal("opaque send changed")
				}
				if body["msg_type"] == float64(2) {
					if body["content"] != nil || body["markdown"].(map[string]interface{})["content"] != "**hello**" || body["keyboard"].(map[string]interface{})["id"] != "template" {
						t.Fatal(body)
					}
				}
			}
		})
	}
}
func TestProtocolSyncErrorsAndMalformedSuccess(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		upload bool
	}{{"unauthorized", 401, `{"code":11244,"message":"secret-token"}`, false}, {"rate", 429, `{"code":22009}`, false}, {"business", 200, `{"err_code":987654,"message":"private-body"}`, false}, {"empty-send", 200, `{}`, false}, {"empty-upload", 200, `{}`, true}, {"non-json", 502, `private-body`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var count int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&count, 1)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Tps-Trace-ID", "trace-error")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			api := localAPI(t, server, nil)
			var message dto.APIMessage = &dto.MessageToCreate{Content: "private", MsgType: dto.TextMsg, MsgID: "m", MsgSeq: 1}
			if tc.upload {
				message = &dto.RichMediaMessage{FileType: 1, URL: "https://example.invalid"}
			}
			_, err := api.PostC2CMessage(context.Background(), "user", message)
			if err == nil || strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "private-body") || atomic.LoadInt32(&count) != 1 {
				t.Fatal("unsafe or retried error", err)
			}
			if tc.name == "unauthorized" || tc.name == "rate" || tc.name == "business" {
				var p *errs.PlatformError
				if !errors.As(err, &p) || p.Status != tc.status || p.TraceID != "trace-error" {
					t.Fatal(err)
				}
				if errs.Error(err).Code() != tc.status {
					t.Fatal("legacy error helper lost HTTP status")
				}
			}
		})
	}
}

type invalidatingTokens struct{ invalidated int32 }

func (s *invalidatingTokens) Token() (*oauth2.Token, error) {
	value := "stale"
	if atomic.LoadInt32(&s.invalidated) > 0 {
		value = "fresh"
	}
	return &oauth2.Token{TokenType: "QQBot", AccessToken: value}, nil
}
func (s *invalidatingTokens) Invalidate() { atomic.AddInt32(&s.invalidated, 1) }
func TestProtocolSyncTokenRecoveryWithoutReplay(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&count, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.WriteHeader(401)
			io.WriteString(w, `{"code":11244}`)
			return
		}
		if r.Header.Get("Authorization") != "QQBot fresh" {
			t.Error("token not invalidated")
		}
		io.WriteString(w, `{"id":"sent"}`)
	}))
	defer server.Close()
	tokens := &invalidatingTokens{}
	api := localAPI(t, server, tokens)
	if _, err := api.PostC2CMessage(context.Background(), "user", &dto.MessageToCreate{MsgID: "m", MsgSeq: 1}); err == nil {
		t.Fatal("expired token accepted")
	}
	if atomic.LoadInt32(&count) != 1 || atomic.LoadInt32(&tokens.invalidated) != 1 {
		t.Fatal("non-idempotent POST replayed")
	}
	if _, err := api.PostC2CMessage(context.Background(), "user", &dto.MessageToCreate{MsgID: "m", MsgSeq: 2}); err != nil {
		t.Fatal(err)
	}
}
func TestProtocolSyncConcurrentRequestTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.Split(r.URL.Path, "/")[3]
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Tps-Trace-ID", name)
		if strings.HasPrefix(name, "error-") {
			w.WriteHeader(403)
			io.WriteString(w, `{"code":999123}`)
		} else {
			io.WriteString(w, `{"id":"sent"}`)
		}
	}))
	defer server.Close()
	api := localAPI(t, server, nil)
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			prefix := "success-"
			if n%2 == 1 {
				prefix = "error-"
			}
			name := fmt.Sprintf("%s%d", prefix, n)
			message, err := api.PostGroupMessage(context.Background(), name, &dto.MessageToCreate{MsgType: dto.TextMsg, MsgID: "m", MsgSeq: 1})
			if n%2 == 1 {
				var p *errs.PlatformError
				if !errors.As(err, &p) || p.TraceID != name {
					t.Error("error correlation crossed requests")
				}
			} else if err != nil || message.TraceID != name {
				t.Error("success correlation crossed requests")
			}
			_ = api.TraceID() // legacy getter must also be race-free, but is last-completed only.
		}(i)
	}
	wg.Wait()
}
func TestProtocolSyncCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"sent"}`)
	}))
	defer server.Close()
	api := localAPI(t, server, nil).WithTimeout(10 * time.Millisecond)
	if _, err := api.PostC2CMessage(context.Background(), "user", &dto.MessageToCreate{MsgID: "m", MsgSeq: 1}); err == nil {
		t.Fatal("timeout accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.PostGroupMessage(ctx, "group", &dto.MessageToCreate{}); err == nil {
		t.Fatal("cancellation ignored")
	}
}

type capturedLogger struct {
	mu   sync.Mutex
	text strings.Builder
}

func (l *capturedLogger) add(v string)                      { l.mu.Lock(); defer l.mu.Unlock(); l.text.WriteString(v) }
func (l *capturedLogger) Debug(v ...interface{})            { l.add(fmt.Sprint(v...)) }
func (l *capturedLogger) Info(v ...interface{})             { l.add(fmt.Sprint(v...)) }
func (l *capturedLogger) Warn(v ...interface{})             { l.add(fmt.Sprint(v...)) }
func (l *capturedLogger) Error(v ...interface{})            { l.add(fmt.Sprint(v...)) }
func (l *capturedLogger) Debugf(f string, v ...interface{}) { l.add(fmt.Sprintf(f, v...)) }
func (l *capturedLogger) Infof(f string, v ...interface{})  { l.add(fmt.Sprintf(f, v...)) }
func (l *capturedLogger) Warnf(f string, v ...interface{})  { l.add(fmt.Sprintf(f, v...)) }
func (l *capturedLogger) Errorf(f string, v ...interface{}) { l.add(fmt.Sprintf(f, v...)) }
func (l *capturedLogger) Sync() error                       { return nil }
func TestProtocolSyncDebugLogsExcludeAuthorizationAndBodies(t *testing.T) {
	old := sdklog.DefaultLogger
	defer botgo.SetLogger(old)
	recorder := &capturedLogger{}
	botgo.SetLogger(recorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"secret-response-marker"}`)
	}))
	defer server.Close()
	api := localAPI(t, server, nil).SetDebug(true)
	if _, err := api.PostC2CMessage(context.Background(), "private-target-marker", &dto.MessageToCreate{Content: "private-content-marker", MsgID: "private-id-marker", MsgSeq: 1}); err != nil {
		t.Fatal(err)
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for _, marker := range []string{"fake-token", "Authorization", "private-target-marker", "private-content-marker", "private-id-marker", "secret-response-marker"} {
		if strings.Contains(recorder.text.String(), marker) {
			t.Fatal("SDK log exposed private marker", marker)
		}
	}
}
