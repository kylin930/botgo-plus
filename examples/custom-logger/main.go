package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kylin930/botgo-plus"
	"github.com/kylin930/botgo-plus/dto"
	"github.com/kylin930/botgo-plus/dto/message"
	"github.com/kylin930/botgo-plus/event"
	"github.com/kylin930/botgo-plus/interaction/webhook"
	"github.com/kylin930/botgo-plus/openapi"
	"github.com/kylin930/botgo-plus/token"
	"gopkg.in/yaml.v3"
)

const (
	host_ = "0.0.0.0"
	port_ = 9000
	path_ = "/qqbot"
)

func main() {
	// 初始化新的文件 logger，并使用相对路径来作为日志存放位置，设置最小日志界别为 DebugLevel
	logger, err := New("./", DebugLevel)
	if err != nil {
		log.Fatalln("error log new", err)
	}
	// 把新的 logger 设置到 sdk 上，替换掉老的控制台 logger
	botgo.SetLogger(logger)
	content, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalln("load config file failed, err:", err)
	}
	credentials := &token.QQBotCredentials{}
	if err = yaml.Unmarshal(content, &credentials); err != nil {
		log.Fatalln("parse config failed, err:", err)
	}

	tokenSource := token.NewQQBotTokenSource(credentials)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() //释放刷新协程
	if err = token.StartRefreshAccessToken(ctx, tokenSource); err != nil {
		log.Fatalln(err)
	}
	// 初始化 openapi，正式环境
	api := botgo.NewOpenAPI(credentials.AppID, tokenSource).WithTimeout(5 * time.Second).SetDebug(true)
	// 根据不同的回调，生成 intents
	_ = event.RegisterHandlers(GuildATMessageEventHandler(api))
	http.HandleFunc(path_, func(writer http.ResponseWriter, request *http.Request) {
		webhook.HTTPHandler(writer, request, credentials)
	})
	if err = http.ListenAndServe(fmt.Sprintf("%s:%d", host_, port_), nil); err != nil {
		log.Fatal("setup server fatal:", err)
	}
}

// GuildATMessageEventHandler 实现处理 at 消息的回调
func GuildATMessageEventHandler(api openapi.OpenAPI) event.ATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSATMessageData) error {
		log.Printf("[%s] %s", event.Type, data.Content)
		input := strings.ToLower(message.ETLInput(data.Content))
		log.Printf("clear input content is: %s", input)
		return nil
	}
}
