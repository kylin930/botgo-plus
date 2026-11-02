package event

import (
	"encoding/json"
	"github.com/kylin930/botgo-plus/dto"
	"testing"
)

func TestEnhancedEventRegistrationAndParsing(t *testing.T) {
	old := DefaultHandlers
	defer func() { DefaultHandlers = old }()
	seen := map[string]int{}
	intent := RegisterHandlers(
		GroupMessageEventHandler(func(p *dto.WSPayload, m *dto.WSGroupMessageData) error {
			if m.ID != "message" || m.Author.MemberOpenID != "member" {
				t.Error("group DTO")
			}
			seen["group"]++
			return nil
		}),
		C2CMessageEventHandler(func(p *dto.WSPayload, m *dto.WSC2CMessageData) error {
			if m.Author.UserOpenID != "user" {
				t.Error("C2C DTO")
			}
			seen["c2c"]++
			return nil
		}),
		GroupManageEventHandler(func(p *dto.WSPayload, m *dto.WSGroupManageData) error {
			if m.GroupOpenID != "group" || m.OpMemberOpenID != "member" {
				t.Error("group management DTO")
			}
			seen["group-manage"]++
			return nil
		}),
		C2CManageEventHandler(func(p *dto.WSPayload, m *dto.WSC2CManageData) error {
			if m.OpenID != "user" {
				t.Error("C2C management DTO")
			}
			seen["c2c-manage"]++
			return nil
		}),
		SubscribeMsgStatusEventHandler(func(p *dto.WSPayload, m *dto.WSSubscribeMsgStatus) error { seen["subscribe"]++; return nil }),
	)
	if intent != dto.IntentGroupMessages {
		t.Fatal(intent)
	}
	for _, kind := range []dto.EventType{dto.EventGroupMessageCreate, dto.EventC2CMessageCreate, dto.EventGroupAddRobot, dto.EventGroupDelRobot, dto.EventGroupMsgReject, dto.EventGroupMsgReceive, dto.EventC2CMsgReject, dto.EventC2CMsgReceive, dto.EventSubscribeMsgStatus} {
		raw, _ := json.Marshal(map[string]interface{}{"op": 0, "t": kind, "d": map[string]interface{}{"id": "message", "group_openid": "group", "op_member_openid": "member", "openid": "user", "author": map[string]string{"member_openid": "member", "user_openid": "user"}}})
		var payload dto.WSPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		payload.RawMessage = raw
		if err := ParseAndHandle(&payload); err != nil {
			t.Fatal(err)
		}
	}
	if seen["group"] != 1 || seen["c2c"] != 1 || seen["group-manage"] != 4 || seen["c2c-manage"] != 2 || seen["subscribe"] != 1 {
		t.Fatal(seen)
	}
}
