package dto

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestUserAndMessageCompatibility(t *testing.T) {
	raw := []byte(`{"id":"message","seq":12,"author":{"id":"guild","member_openid":"member","user_openid":"user","is_you":true,"member_role":"admin","scope":"group","union_openid":"union","union_user_account":"account","status":1},"timestamp":123}`)
	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		t.Fatal(err)
	}
	user := message.Author
	if message.ID != "message" || message.Seq != 12 || user.MemberOpenID != "member" || user.UserOpenID != "user" || !user.IsYou || user.MemberRole != "admin" || user.Scope != "group" || user.UnionOpenID != "union" || user.UnionUserAccount != "account" || user.Status != 1 {
		t.Fatalf("public fields lost: %+v", user)
	}
	for field, tag := range map[string]string{"IsYou": "is_you", "MemberRole": "member_role", "Scope": "scope", "MemberOpenID": "member_openid", "UserOpenID": "user_openid"} {
		f, ok := reflect.TypeOf(User{}).FieldByName(field)
		if !ok || f.Tag.Get("json") != tag {
			t.Fatal("incompatible JSON field", field)
		}
	}
	// Compile-check original enhanced fields/types; replacing whole DTOs loses these.
	_ = MessageToCreate{MsgID: "m", EventID: "e", MsgSeq: 2, Stream: &Stream{State: 1, ID: "stream", Index: 1, Reset: true}, PromptKeyboard: &PromptKeyboard{}, ActionButton: &ActionButton{Feedback: true}, FeatureID: 1, SubscribeID: "sub", InputNotify: &InputNotify{}}
	_ = MediaInfo{FileInfo: []byte("legacy"), FileUUID: "uuid", TTL: 300}
	_ = Message{MessageScene: MessageScene{}, FileInfo: []byte("legacy")}
}
func TestTimestampVariants(t *testing.T) {
	for _, raw := range []string{`123`, `"123"`, `"1970-01-01T00:02:03Z"`} {
		var stamp Timestamp
		if err := json.Unmarshal([]byte(raw), &stamp); err != nil {
			t.Fatal(err)
		}
		when, err := stamp.Time()
		if err != nil || !when.Equal(time.Unix(123, 0)) {
			t.Fatal(stamp, err)
		}
	}
	for _, raw := range []string{`1.5`, `true`, `{}`} {
		var stamp Timestamp
		if json.Unmarshal([]byte(raw), &stamp) == nil {
			t.Fatal("invalid timestamp accepted", raw)
		}
	}
	var stamp Timestamp = "old"
	if json.Unmarshal([]byte(`null`), &stamp) != nil || stamp != "" {
		t.Fatal("null timestamp")
	}
}
func TestOpaqueFileInfo(t *testing.T) {
	for _, opaque := range []string{"opaque/not-base64!", "b3BhcXVl", "opaque+==%"} {
		encoded, _ := json.Marshal(map[string]string{"file_info": opaque})
		var response Message
		if err := json.Unmarshal(encoded, &response); err != nil {
			t.Fatal(err)
		}
		if response.FileInfoRaw != opaque {
			t.Fatal("opaque response changed")
		}
		data, err := json.Marshal(MediaInfo{FileInfo: response.FileInfo, FileInfoRaw: response.FileInfoRaw})
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]interface{}
		json.Unmarshal(data, &fields)
		if fields["file_info"] != opaque {
			t.Fatal("opaque outgoing changed", string(data))
		}
	}
}
func TestExtendedIntentAssociations(t *testing.T) {
	for _, event := range []EventType{EventGroupMessageCreate, EventGroupAtMessageCreate, EventC2CMessageCreate, EventSubscribeMsgStatus, EventC2CFriendAdd, EventC2CFriendDel, EventGroupAddRobot, EventGroupDelRobot, EventGroupMsgReject, EventGroupMsgReceive, EventC2CMsgReject, EventC2CMsgReceive} {
		if EventToIntent(event) != IntentGroupMessages {
			t.Fatal("missing Intent", event)
		}
	}
	if EventToIntent(EventEnterAIO) != IntentEnterAIO || IntentOpenForum != 1<<18 || IntentAudioOrLiveChannelMember != 1<<19 {
		t.Fatal("extended Intent lost")
	}
}
