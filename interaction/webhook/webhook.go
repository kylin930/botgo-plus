// Package webhook HTTP回调处理
package webhook

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"

	"github.com/kylin930/botgo-plus/dto"
	"github.com/kylin930/botgo-plus/event"
	"github.com/kylin930/botgo-plus/interaction/signature"
	"github.com/kylin930/botgo-plus/log"
	"github.com/kylin930/botgo-plus/token"
)

type ack struct {
	Op   dto.OPCode `json:"op"`
	Data uint32     `json:"d"`
}

// GenHeartbeatACK 生成 http gateway 的心跳回包
func GenHeartbeatACK(seq uint32) string {
	s, _ := json.Marshal(ack{Op: dto.WSHeartbeatAck, Data: seq})
	return string(s)
}

// GenDispatchACK 生成事件包的回包，如果处理失败，则返回的 d 为 1，服务端会尝试重试
func GenDispatchACK(success bool) string {
	var r uint32
	if !success {
		r = 1
	}
	s, _ := json.Marshal(ack{Op: dto.HTTPCallbackAck, Data: r})
	return string(s)
}

// Deprecated: DefaultGetSecretFunc 默认的获取 secret 的函数，默认从环境变量读取
// 开发者如果需要从自己的配置文件，或者是其他地方获取 secret，可以重写这个函数
var DefaultGetSecretFunc = func() string {
	return os.Getenv("QQBotSecret")
}

// HTTPHandler 用户处理回调时间，该函数实现的是 https://pkg.go.dev/net/http#HandleFunc 所要求的 handler
// 会自动进行签名验证，心跳包回复，以及根据使用 event.RegisterHandlers 注册的 handler 去执行不同的 handler 来处理事件
// 如果开发者不想在接收事件的地方处理，可以实现 DefaultHandlers.Plain 然后在内部处理相关的异步生产或者转发的逻辑
func HTTPHandler(w http.ResponseWriter, r *http.Request, credentials *token.QQBotCredentials) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if credentials == nil || credentials.AppSecret == "" {
		http.Error(w, "webhook credentials unavailable", 503)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		http.Error(w, "invalid callback body", 400)
		return
	}
	pass, err := signature.Verify(credentials.AppSecret, r.Header, body)
	if err != nil || !pass {
		http.Error(w, "invalid signature", 401)
		return
	}
	payload := &dto.WSPayload{}
	if json.Unmarshal(body, payload) != nil {
		http.Error(w, "invalid payload", 400)
		return
	}
	payload.RawMessage = body
	payload.Session = &dto.Session{AppID: credentials.AppID}
	w.Header().Set("Content-Type", "application/json")
	if payload.OPCode == dto.HTTPCallbackValidation {
		data, ok := payload.Data.(map[string]interface{})
		if !ok {
			http.Error(w, "invalid validation", 400)
			return
		}
		plain, plainOK := data["plain_token"].(string)
		ts, tsOK := data["event_ts"].(string)
		if !plainOK || !tsOK || plain == "" || ts == "" {
			http.Error(w, "invalid validation", 400)
			return
		}
		response := GenValidationACK(&dto.WHValidationReq{PlainToken: plain, EventTs: ts}, r.Header, credentials.AppSecret)
		if response == nil {
			http.Error(w, "validation failed", 400)
			return
		}
		_, _ = w.Write(response)
		return
	}
	result := parsePayload(payload, "")
	if result == "" {
		http.Error(w, "unsupported or invalid callback", 400)
		return
	}
	_, _ = io.WriteString(w, result)
}

func parsePayload(payload *dto.WSPayload, traceID string) string {
	// 处理心跳包
	if payload.OPCode == dto.WSHeartbeat {
		seq, ok := payload.Data.(float64)
		if !ok || seq < 0 || seq > math.MaxUint32 || seq != math.Trunc(seq) {
			return ""
		}
		return GenHeartbeatACK(uint32(seq))
	}
	// 处理事件
	if payload.OPCode == dto.WSDispatchEvent {
		// 解析具体事件，并投递给业务注册的 handler
		if err := event.ParseAndHandle(payload); err != nil {
			log.Error("webhook event handler failed")
			return GenDispatchACK(false)
		}
		return GenDispatchACK(true)
	}

	return ""
}

// GenValidationACK 生成回调校验回包
func GenValidationACK(req *dto.WHValidationReq, header http.Header, secret string) []byte {
	if req == nil || req.PlainToken == "" || req.EventTs == "" {
		return nil
	}
	h := header.Clone()
	h.Set(signature.HeaderTimestamp, req.EventTs)
	sig, err := signature.Generate(secret, h, []byte(req.PlainToken))
	if err != nil {
		log.Errorf("generate signature failed:%+v", err)
		return nil
	}
	rsp, err := json.Marshal(
		&dto.WHValidationRsp{
			PlainToken: req.PlainToken,
			Signature:  sig,
		})
	if err != nil {
		log.Error("validation encoding failed")
		return nil
	}
	return rsp
}
