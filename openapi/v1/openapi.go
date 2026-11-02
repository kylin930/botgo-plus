// Package v1 是 openapi v1 版本的实现。
package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2" // resty 是一个优秀的 rest api 客户端，可以极大的减少开发基于 rest 标准接口求请求的封装工作量
	"github.com/kylin930/botgo-plus/constant"
	"github.com/kylin930/botgo-plus/errs"
	"github.com/kylin930/botgo-plus/log"
	"github.com/kylin930/botgo-plus/openapi"
	"github.com/kylin930/botgo-plus/version"
	"golang.org/x/oauth2"
)

// MaxIdleConns 默认指定空闲连接池大小
const MaxIdleConns = 3000

type openAPI struct {
	appID       string
	tokenSource oauth2.TokenSource
	timeout     time.Duration

	sandbox     bool   // 请求沙箱环境
	debug       bool   // debug 模式，调试sdk时候使用
	lastTraceID string // lastTraceID id
	traceMu     sync.Mutex
	baseURL     string

	restyClient *resty.Client // resty client 复用
}

// Setup 注册
func Setup() {
	openapi.Register(openapi.APIv1, &openAPI{})
}

// Version 创建当前版本
func (o *openAPI) Version() openapi.APIVersion {
	return openapi.APIv1
}

// TraceID 获取 lastTraceID id
func (o *openAPI) TraceID() string {
	o.traceMu.Lock()
	defer o.traceMu.Unlock()
	return o.lastTraceID
}

// Setup 生成一个实例
func (o *openAPI) Setup(botAppID string, tokenSource oauth2.TokenSource, inSandbox bool) openapi.OpenAPI {
	api := &openAPI{
		appID:       botAppID,
		tokenSource: tokenSource,
		timeout:     5 * time.Second,
		sandbox:     inSandbox,
	}
	api.setupClient(botAppID) // 初始化可复用的 client
	return api
}

// WithTimeout 设置请求接口超时时间
func (o *openAPI) WithTimeout(duration time.Duration) openapi.OpenAPI {
	o.restyClient.SetTimeout(duration)
	return o
}

// SetDebug retains the legacy switch; diagnostics contain metadata only, never raw HTTP dumps.
func (o *openAPI) SetDebug(debug bool) openapi.OpenAPI {
	o.debug = debug
	o.restyClient.Debug = false // metadata only; never dump credentials or message bodies
	return o
}

// Transport 透传请求
func (o *openAPI) Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	resp, err := o.request(ctx).SetBody(body).Execute(method, url)
	if resp == nil {
		return nil, err
	}
	return resp.Body(), err
}

// 初始化 client
func (o *openAPI) setupClient(appID string) {
	o.restyClient = resty.New().
		SetTransport(createTransport(nil, MaxIdleConns)). // 自定义 transport
		SetLogger(log.DefaultLogger).
		SetDebug(false).
		SetTimeout(o.timeout).
		SetHeader("User-Agent", version.String()).
		SetHeader("X-Union-Appid", appID).
		SetPreRequestHook(
			func(_ *resty.Client, request *http.Request) error {
				// 执行请求前过滤器
				// 由于在 `OnBeforeRequest` 的时候，request 还没生成，所以 filter 不能使用，所以放到 `PreRequestHook`
				return openapi.DoReqFilterChains(request, nil)
			},
		).
		OnBeforeRequest(
			func(_ *resty.Client, request *resty.Request) error {
				tk, err := o.tokenSource.Token()
				if err != nil {
					return errors.New("QQ authentication token unavailable")
				}
				request.SetAuthScheme(tk.TokenType)
				log.Debugf("token type:%s", tk.TokenType)
				request.SetAuthToken(tk.AccessToken)
				return nil
			},
		).
		// 设置请求之后的钩子，打印日志，判断状态码
		OnAfterResponse(
			func(_ *resty.Client, resp *resty.Response) error {
				log.Infof("%v", respInfo(resp))
				// 执行请求后过滤器
				if err := openapi.DoRespFilterChains(resp.Request.RawRequest, resp.RawResponse); err != nil {
					return err
				}
				traceID := safeTraceID(resp.Header().Get(constant.HeaderTraceID))
				o.traceMu.Lock()
				o.lastTraceID = traceID
				o.traceMu.Unlock()
				var body errBody
				_ = json.Unmarshal(resp.Body(), &body)
				code := body.Code
				if body.ErrCode != 0 {
					code = body.ErrCode
				}
				if !openapi.IsSuccessStatus(resp.StatusCode()) || code != 0 {
					if code == errs.APICodeTokenExpireOrNotExist {
						if source, ok := o.tokenSource.(interface{ Invalidate() }); ok {
							source.Invalidate()
						}
					}
					return &errs.PlatformError{Status: resp.StatusCode(), Code: code, TraceID: traceID}
				}
				return nil
			},
		)
}

// request 每个请求，都需要创建一个 request
func (o *openAPI) request(ctx context.Context) *resty.Request {
	return o.restyClient.R().SetContext(ctx)
}

// GetAppID 获取接口地址，会处理沙箱环境判断
func (o *openAPI) GetAppID() string {
	if o == nil {
		return ""
	}
	return o.appID
}

// errBody 请求出错情况下的body结构
type errBody struct {
	Message string `json:"message"`  // 错误原因
	Code    int    `json:"code"`     // 错误码，后续废弃
	ErrCode int    `json:"err_code"` // 错误码
	TraceID string `json:"trace_id"` // 服务端traceID, 用于问题排查
}

// respInfo intentionally excludes URL targets, headers, request and response bodies.
func respInfo(resp *resty.Response) string {
	return fmt.Sprintf("[OPENAPI] method=%s status=%d elapsed=%s", resp.Request.Method, resp.StatusCode(), resp.Time())
}

// WithBaseURL is an optional SDK extension for local HTTP integration tests.
// Configure before using the instance concurrently; application config cannot set it.
func (o *openAPI) WithBaseURL(url string) openapi.OpenAPI { o.baseURL = url; return o }

func createTransport(localAddr net.Addr, idleConns int) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          idleConns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   idleConns,
		MaxConnsPerHost:       idleConns,
	}
}

// Correlation IDs are bounded metadata, not an arbitrary response/header dump.
func safeTraceID(value string) string {
	if len(value) > 128 {
		return ""
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	}) >= 0 {
		return ""
	}
	return value
}
