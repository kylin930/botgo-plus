<div align="center">

# botgo-plus

_腾讯 QQ 机器人官方 Go SDK 增强版，补齐群聊与 C2C 场景的完整支持_

</div>

<p align="center">
  <a href="https://raw.githubusercontent.com/kylin930/botgo-plus/main/LICENSE">
    <img src="https://img.shields.io/github/license/kylin930/botgo-plus" alt="license">
  </a>
  <a href="https://github.com/kylin930/botgo-plus/releases">
    <img src="https://img.shields.io/github/v/release/kylin930/botgo-plus?color=blueviolet&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/tencent-connect/botpy">
    <img src="https://img.shields.io/badge/botpy-aligned-blue?style=flat" alt="botpy">
  </a>
  <a href="https://goreportcard.com/report/github.com/kylin930/botgo-plus">
    <img src="https://goreportcard.com/badge/github.com/kylin930/botgo-plus" alt="GoReportCard">
  </a>
  <a href="https://pkg.go.dev/github.com/kylin930/botgo-plus">
    <img src="https://pkg.go.dev/badge/github.com/kylin930/botgo-plus.svg" alt="Go Reference">
  </a>
</p>

<p align="center">
  <a href="#安装">安装</a>
  ·
  <a href="#快速开始">快速开始</a>
  ·
  <a href="#事件列表">事件列表</a>
  ·
  <a href="#魔改特性来自-gensokyo">魔改特性</a>
</p>

---

基于官方 [botgo](https://github.com/tencent-connect/botgo) 魔改，参考官方 [botpy](https://github.com/tencent-connect/botpy)（Python SDK）的完整实现进行对齐，
补齐群聊（Group）与 C2C（好友私聊）场景下缺失的事件、字段和数据结构。

---

## 与官方 botgo 的差异

官方 botgo SDK 主要围绕频道（Guild）场景设计，在群聊和 C2C 场景下存在以下缺失：

| 对比项 | 官方 botgo | botgo-plus |
|--------|-----------|------------|
| 群聊消息发送者身份 (`member_openid`) | 缺失 | 支持 |
| C2C 消息发送者身份 (`user_openid`) | 缺失 | 支持 |
| 机器人加入群聊事件 (`GROUP_ADD_ROBOT`) | 缺失 | 支持 |
| 机器人退出群聊事件 (`GROUP_DEL_ROBOT`) | 缺失 | 支持 |
| 群主动消息权限变更事件 (`GROUP_MSG_REJECT/RECEIVE`) | 缺失 | 支持 |
| C2C 主动消息权限变更事件 (`C2C_MSG_REJECT/RECEIVE`) | 缺失 | 支持 |
| 全局消息序号 (`seq`) | 缺失 | 支持 |
| 用户状态 (`status`) | 缺失 | 支持 |
| 富媒体 FileUUID / TTL 字段 | 缺失 | 支持 |
| 开放论坛事件 Intent | 缺失 | 支持 |
| 音视频/直播子频道 Intent | 缺失 | 支持 |

---

## 安装

```bash
go get github.com/kylin930/botgo-plus
```

---

## 快速开始

### 1. 基础配置与服务启动

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    botgo "github.com/kylin930/botgo-plus"
    "github.com/kylin930/botgo-plus/dto"
    "github.com/kylin930/botgo-plus/event"
    "github.com/kylin930/botgo-plus/interaction/webhook"
    "github.com/kylin930/botgo-plus/token"
)

func main() {
    // 初始化凭证
    credentials := &token.QQBotCredentials{
        AppID:     "你的AppID",
        AppSecret: "你的AppSecret",
    }

    // 创建 Token 并启动自动刷新
    tokenSource := token.NewQQBotTokenSource(credentials)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := token.StartRefreshAccessToken(ctx, tokenSource); err != nil {
        log.Fatalln(err)
    }

    // 初始化 OpenAPI
    api := botgo.NewOpenAPI(credentials.AppID, tokenSource).WithTimeout(5 * time.Second)

    // 注册事件处理器
    _ = event.RegisterHandlers(
        GroupATMessageEventHandler(api),
        C2CMessageEventHandler(api),
        GroupManageEventHandler(),
        C2CManageEventHandler(),
    )

    // 启动 Webhook 回调服务
    http.HandleFunc("/qqbot", func(w http.ResponseWriter, r *http.Request) {
        webhook.HTTPHandler(w, r, credentials)
    })
    log.Println("服务启动于 :9000")
    http.ListenAndServe(":9000", nil)
}
```

### 2. 群聊消息处理

```go
func GroupATMessageEventHandler(api openapi.OpenAPI) event.GroupATMessageEventHandler {
    return func(evt *dto.WSPayload, data *dto.WSGroupATMessageData) error {
        // 获取发送者 openid（官方 botgo 拿不到这个字段）
        senderOpenID := data.Author.MemberOpenID
        log.Printf("[群聊] %s 在群 %s 说: %s", senderOpenID, data.GroupID, data.Content)

        // 回复消息
        _, err := api.PostGroupMessage(
            context.Background(),
            data.GroupID,
            &dto.MessageToCreate{
                Content: "收到你的消息啦",
                MsgID:   data.ID,
                MsgType: dto.TextMsg,
            },
        )
        return err
    }
}
```

### 3. C2C（好友私聊）消息处理

```go
func C2CMessageEventHandler(api openapi.OpenAPI) event.C2CMessageEventHandler {
    return func(evt *dto.WSPayload, data *dto.WSC2CMessageData) error {
        // 获取发送者 openid（官方 botgo 拿不到这个字段）
        senderOpenID := data.Author.UserOpenID
        log.Printf("[C2C] %s 说: %s", senderOpenID, data.Content)

        // 回复消息
        _, err := api.PostC2CMessage(
            context.Background(),
            senderOpenID,
            &dto.MessageToCreate{
                Content: "你好",
                MsgID:   data.ID,
                MsgType: dto.TextMsg,
            },
        )
        return err
    }
}
```

### 4. 群管理事件处理

```go
func GroupManageEventHandler() event.GroupManageEventHandler {
    return func(evt *dto.WSPayload, data *dto.WSGroupManageData) error {
        switch evt.Type {
        case dto.EventGroupAddRobot:
            log.Printf("机器人被加入群聊: %s (操作人: %s)", data.GroupOpenID, data.OpMemberOpenID)
        case dto.EventGroupDelRobot:
            log.Printf("机器人被移出群聊: %s (操作人: %s)", data.GroupOpenID, data.OpMemberOpenID)
        case dto.EventGroupMsgReject:
            log.Printf("群 %s 关闭了主动消息", data.GroupOpenID)
        case dto.EventGroupMsgReceive:
            log.Printf("群 %s 开启了主动消息", data.GroupOpenID)
        }
        return nil
    }
}
```

### 5. C2C 管理事件处理

```go
func C2CManageEventHandler() event.C2CManageEventHandler {
    return func(evt *dto.WSPayload, data *dto.WSC2CManageData) error {
        switch evt.Type {
        case dto.EventC2CMsgReject:
            log.Printf("用户 %s 关闭了主动消息", data.OpenID)
        case dto.EventC2CMsgReceive:
            log.Printf("用户 %s 开启了主动消息", data.OpenID)
        }
        return nil
    }
}
```

---

## 事件列表

### 群聊相关事件

| 事件常量 | 事件类型字符串 | 说明 | Handler 类型 |
|---------|--------------|------|-------------|
| `EventGroupAtMessageCreate` | `GROUP_AT_MESSAGE_CREATE` | 群中@机器人消息 | `GroupATMessageEventHandler` |
| `EventGroupAddRobot` | `GROUP_ADD_ROBOT` | 机器人加入群聊 | `GroupManageEventHandler` |
| `EventGroupDelRobot` | `GROUP_DEL_ROBOT` | 机器人退出群聊 | `GroupManageEventHandler` |
| `EventGroupMsgReject` | `GROUP_MSG_REJECT` | 群关闭主动消息 | `GroupManageEventHandler` |
| `EventGroupMsgReceive` | `GROUP_MSG_RECEIVE` | 群开启主动消息 | `GroupManageEventHandler` |

### C2C（好友私聊）相关事件

| 事件常量 | 事件类型字符串 | 说明 | Handler 类型 |
|---------|--------------|------|-------------|
| `EventC2CMessageCreate` | `C2C_MESSAGE_CREATE` | C2C 消息 | `C2CMessageEventHandler` |
| `EventC2CFriendAdd` | `FRIEND_ADD` | 添加机器人好友 | `C2CFriendEventHandler` |
| `EventC2CFriendDel` | `FRIEND_DEL` | 删除机器人好友 | `C2CFriendEventHandler` |
| `EventC2CMsgReject` | `C2C_MSG_REJECT` | 用户关闭主动消息 | `C2CManageEventHandler` |
| `EventC2CMsgReceive` | `C2C_MSG_RECEIVE` | 用户开启主动消息 | `C2CManageEventHandler` |

---

## 字段补充说明

### User 结构体新增字段

原官方 botgo 的 `User` 结构体仅包含频道场景的字段，本版本补充了群聊/C2C 场景所需的 openid 字段：

```go
type User struct {
    ID               string `json:"id"`               // 频道用户ID
    Username         string `json:"username"`
    Avatar           string `json:"avatar"`
    Bot              bool   `json:"bot"`
    Status           int    `json:"status"`           // 新增: 用户状态
    MemberOpenID     string `json:"member_openid"`    // 新增: 群聊成员 openid
    UserOpenID       string `json:"user_openid"`      // 新增: C2C 用户 openid
    // ... 其他字段
}
```

在不同场景下获取发送者身份的方式：

- 频道场景：`data.Author.ID`
- 群聊场景：`data.Author.MemberOpenID`
- C2C 场景：`data.Author.UserOpenID`

### MediaInfo 结构体新增字段

```go
type MediaInfo struct {
    FileUUID string `json:"file_uuid,omitempty"` // 新增: 文件ID
    FileInfo []byte `json:"file_info,omitempty"` // 富媒体文件信息
    TTL      int    `json:"ttl,omitempty"`       // 新增: 有效期（秒）
}
```

---

## 频道与群聊/C2C 的差异说明

官方 API 在不同场景下的字段设计存在差异，开发时请注意：

| 对比项 | 频道 (Guild) | 群聊 (Group) | C2C |
|--------|-------------|-------------|-----|
| 用户标识字段 | `author.id` | `author.member_openid` | `author.user_openid` |
| content 中 @格式 | `<@!userid>` | 纯文本，无特殊包裹 | 纯文本，无特殊包裹 |
| mentions 数组 | 返回 | 可能不返回 | 可能不返回 |
| mention_everyone | 返回 | 不返回 | 不返回 |

如果需要确认官方在群聊/C2C 场景下是否返回 `mentions`，可以在 Handler 中打印原始 payload：

```go
import "github.com/tidwall/gjson"

func(evt *dto.WSPayload, data *dto.WSGroupATMessageData) error {
    mentions := gjson.Get(string(evt.RawMessage), "d.mentions")
    log.Printf("mentions 原始内容: %s", mentions.Raw)
    return nil
}
```

---

## 魔改特性（来自 gensokyo）

除了对齐 botpy 的完整功能外，本版本还从 [gensokyo](https://github.com/Hoshinonyaruko/Gensokyo) 框架中吸收了以下实用增强。

### 富媒体文件直传（无需公网 URL）

官方 SDK 要求富媒体文件先上传到对象存储，再通过 URL 发送。本版本支持直接传递二进制数据：

```go
// 直接上传本地图片文件
imgData, err := os.ReadFile("image.png")
if err != nil {
    return err
}

msg := &dto.RichMediaMessage{
    FileType:   1,       // 1:图片, 2:视频, 3:语音
    FileData:   imgData, // 直接传二进制数据
    SrvSendMsg: true,    // 直接发送（为 true 时占用主动消息频率）
}
_, err = api.PostGroupMessage(ctx, groupID, msg)
```

---

## 变更文件清单

以下是相对于官方 botgo 的所有改动文件：

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `dto/websocket_event.go` | 修改 | 新增 6 个事件类型常量，更新 Intent 映射 |
| `dto/websocket_intent.go` | 修改 | 新增 `IntentOpenForum` 和 `IntentAudioOrLiveChannelMember` |
| `dto/websocket_payload.go` | 修改 | 新增 `WSGroupManageData` 和 `WSC2CManageData` |
| `dto/manage.go` | 新增 | 定义 `GroupManageData` 和 `C2CManageData` 结构体 |
| `dto/user.go` | 修改 | 新增 `Status`、`MemberOpenID`、`UserOpenID` 字段 |
| `dto/message.go` | 修改 | 新增 `Seq` 字段 |
| `dto/message_create.go` | 修改 | `MediaInfo` 新增 `FileUUID`、`TTL` 字段 |
| `event/register.go` | 修改 | 新增 `GroupManageEventHandler`、`C2CManageEventHandler` 及注册逻辑 |
| `event/event.go` | 修改 | 新增事件映射及 2 个解析函数 |

---

## 相关链接

- [官方 QQ 机器人文档](https://bot.q.qq.com/wiki/)
- [官方 botgo 仓库](https://github.com/tencent-connect/botgo)
- [官方 botpy 仓库](https://github.com/tencent-connect/botpy)

---

## 致谢

- [botgo](https://github.com/tencent-connect/botgo) - 腾讯 QQ 机器人官方 Go SDK
- [botpy](https://github.com/tencent-connect/botpy) - 腾讯 QQ 机器人官方 Python SDK，本项目事件与字段对齐的参考来源
- [gensokyo](https://github.com/Hoshinonyaruko/Gensokyo) - 魔改特性来源

---

## License

遵循上游项目的 License。
