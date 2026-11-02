package botgo_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	botgo "github.com/kylin930/botgo-plus"
	"github.com/kylin930/botgo-plus/dto"
	sdklog "github.com/kylin930/botgo-plus/log"
	"github.com/kylin930/botgo-plus/openapi"
	"golang.org/x/oauth2"
)

// Example_mediaReply uses only a local server. Production applications get the
// real incoming message ID and sequence from their per-event reply context.
func Example_mediaReply() {
	old := sdklog.DefaultLogger
	defer botgo.SetLogger(old)
	botgo.SetLogger(exampleLogger{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/files") {
			io.WriteString(w, `{"file_info":"opaque-file-info!"}`)
		} else {
			io.WriteString(w, `{"id":"local-reply-id"}`)
		}
	}))
	defer server.Close()
	api := botgo.NewOpenAPI("local-app", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake", TokenType: "QQBot"}))
	api = api.(interface{ WithBaseURL(string) openapi.OpenAPI }).WithBaseURL(server.URL)
	ctx := context.Background()
	upload, err := api.PostGroupMessage(ctx, "local-group", &dto.RichMediaMessage{FileType: 1, URL: "https://example.invalid/image", SrvSendMsg: false})
	if err != nil {
		panic(err)
	}
	reply, err := api.PostGroupMessage(ctx, "local-group", &dto.MessageToCreate{
		MsgType: dto.RichMediaMsg, Content: " ", MsgID: "local-incoming-id", MsgSeq: 1,
		Media: &dto.MediaInfo{FileInfoRaw: upload.FileInfoRaw},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(reply.ID)
	// Output: local-reply-id
}

// This example suppresses optional SDK metadata so its output is deterministic.
type exampleLogger struct{}

func (exampleLogger) Debug(...interface{})          {}
func (exampleLogger) Info(...interface{})           {}
func (exampleLogger) Warn(...interface{})           {}
func (exampleLogger) Error(...interface{})          {}
func (exampleLogger) Debugf(string, ...interface{}) {}
func (exampleLogger) Infof(string, ...interface{})  {}
func (exampleLogger) Warnf(string, ...interface{})  {}
func (exampleLogger) Errorf(string, ...interface{}) {}
func (exampleLogger) Sync() error                   { return nil }
