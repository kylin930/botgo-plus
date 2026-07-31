package botgo

import (
	"github.com/kylin930/botgo-plus/dto"
	"github.com/kylin930/botgo-plus/event"
	"github.com/kylin930/botgo-plus/log"
	"github.com/kylin930/botgo-plus/openapi"
	"github.com/kylin930/botgo-plus/websocket"
)

// SetLogger 设置 logger，需要实现 sdk 的 log.Logger 接口
func SetLogger(logger log.Logger) {
	log.DefaultLogger = logger
}

// SetSessionManager 注册自己实现的 session manager
func SetSessionManager(m SessionManager) {
	defaultSessionManager = m
}

// SetWebsocketClient 替换 websocket 实现
func SetWebsocketClient(c websocket.WebSocket) {
	websocket.Register(c)
}

// SetOpenAPIClient 注册 openapi 的不同实现，需要设置版本
func SetOpenAPIClient(v openapi.APIVersion, c openapi.OpenAPI) {
	openapi.Register(v, c)
}

// RegisterDispatchEventHandler 注册回调事件处理器
func RegisterDispatchEventHandler(eventType dto.EventType, f func(event *dto.WSPayload, message []byte) error) {
	event.RegisterHandler(dto.WSDispatchEvent, eventType, f)
}
