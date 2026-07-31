package apitest

import (
	"fmt"
	"testing"

	"github.com/kylin930/botgo-plus/dto/keyboard"

	"github.com/kylin930/botgo-plus/dto"
)

func TestMessage(t *testing.T) {
	t.Run(
		"message list", func(t *testing.T) {
			// 先拉取3条消息
			messages, err := api.Messages(
				ctx, testChannelID, &dto.MessagesPager{
					Limit: "3",
				},
			)
			if err != nil {
				t.Error(err)
			}
			index := make(map[int]string)
			for i, message := range messages {
				index[i] = message.ID
				t.Log(message.ID, message.Author.Username, message.Timestamp)
			}

			// 从上面3条的第二条往前拉取
			messages, err = api.Messages(
				ctx, testChannelID, &dto.MessagesPager{
					Type:  dto.MPTBefore,
					ID:    index[1],
					Limit: "2",
				},
			)
			if err != nil {
				t.Error(err)
			}
			for i, message := range messages {
				if i == 2 && index[2] != message.ID {
					t.Error("before id not match")
				}
				t.Log(message.ID, message.Author.Username, message.Timestamp)
			}

			// 从上面3条的第二条往后拉取
			messages, err = api.Messages(
				ctx, testChannelID, &dto.MessagesPager{
					Type:  dto.MPTAfter,
					ID:    index[1],
					Limit: "2",
				},
			)
			if err != nil {
				t.Error(err)
			}
			for i, message := range messages {
				if i == 0 && index[0] != message.ID {
					t.Error("after id not match")
				}
				t.Log(message.ID, message.Author.Username, message.Timestamp)
			}
			// 从上面3条的第二条环绕拉取
			messages, err = api.Messages(
				ctx, testChannelID, &dto.MessagesPager{
					Type:  dto.MPTAround,
					ID:    index[1],
					Limit: "3",
				},
			)
			if err != nil {
				t.Error(err)
			}
			for i, message := range messages {
				if i == 0 && index[0] != message.ID {
					t.Error("around id not match")
				}
				if i == 2 && index[2] != message.ID {
					t.Error("around id not match")
				}
				t.Log(message.ID, message.Author.Username, message.Timestamp)
			}

			message, err := api.Message(ctx, testChannelID, index[0])
			fmt.Println(message)
		},
	)
}

func TestRetractMessage(t *testing.T) {
	msgID := "109b8a401a1231343431313532313831383136323933383420801e28003081b0f30338cd6040c36048f5e4908e0650b1acf8fa05"
	t.Run(
		"消息撤回", func(t *testing.T) {
			err := api.RetractMessage(ctx, "1049883", msgID)
			if err != nil {
				t.Error(err)
			}
			t.Logf("msg id : %v, is deleted", msgID)
		},
	)
}

func TestMessageReference(t *testing.T) {
	t.Run(
		"引用消息", func(t *testing.T) {
			message, err := api.PostMessage(
				ctx, testChannelID, &dto.MessageToCreate{
					Content: "文本引用消息",
					MessageReference: &dto.MessageReference{
						MessageID:             testMessageID,
						IgnoreGetMessageError: false,
					},
				},
			)
			if err != nil {
				t.Error(err)
			}
			t.Logf("message : %v", message)
		},
	)
}

func TestMarkdownMessage(t *testing.T) {
	t.Run(
		"markdown 消息", func(t *testing.T) {
			message, err := api.PostMessage(
				ctx, testChannelID, &dto.MessageToCreate{
					Markdown: &dto.Markdown{
						TemplateID: testMarkdownTemplateID,
						Params: []*dto.MarkdownParams{
							{
								Key:    "title",
								Values: []string{"标题"},
							},
							{
								Key:    "slice",
								Values: []string{"1", "频道名称<#1146349>", "3"},
							},
							{
								Key:    "image",
								Values: []string{"https://pub.idqqimg.com/pc/misc/files/20191015/32ed5b691a1138ac452a59e42f3f83b5.png"},
							},
							{
								Key:    "link",
								Values: []string{"[🔗我的收藏夹](qq.com)"},
							},
							{
								Key:    "desc",
								Values: []string{"简介"},
							},
						},
					},
				},
			)
			if err != nil {
				t.Error(err)
			}
			t.Logf("message : %v", message)
		},
	)
}

func TestKeyboardMessage(t *testing.T) {
	t.Run(
		"消息按钮组件消息", func(t *testing.T) {
			message, err := api.PostMessage(
				ctx, testChannelID, &dto.MessageToCreate{
					Markdown: &dto.Markdown{
						Content: "# 123 \n 今天是个好天气",
					},
					Keyboard: &keyboard.MessageKeyboard{
						Content: &keyboard.CustomKeyboard{
							Rows: []*keyboard.Row{
								{
									Buttons: []*keyboard.Button{
										{
											ID: "1",
											RenderData: &keyboard.RenderData{
												Label:        "指定身份组可点击",
												VisitedLabel: "点击后按钮上文字",
												Style:        0,
											},
											Action: &keyboard.Action{
												Type: keyboard.ActionTypeAtBot,
												Permission: &keyboard.Permission{
													Type:           keyboard.PermissionTypAll,
													SpecifyRoleIDs: []string{"1"},
												},
												ClickLimit:           10,
												Data:                 "/搜索",
												AtBotShowChannelList: true,
											},
										},
										{
											ID: "2",
											RenderData: &keyboard.RenderData{
												Label:        "指定身份组可点击",
												VisitedLabel: "点击后按钮上文字",
												Style:        0,
											},
											Action: &keyboard.Action{
												Type: keyboard.ActionTypeAtBot,
												Permission: &keyboard.Permission{
													Type:           keyboard.PermissionTypeSpecifyUserIDs,
													SpecifyUserIDs: []string{"9859283702500083161"},
												},
												ClickLimit:           10,
												Data:                 "/搜索",
												AtBotShowChannelList: true,
											},
										},
									},
								},
							},
						},
					},
				},
			)
			if err != nil {
				t.Error(err)
			}
			t.Logf("message : %v", message)
		},
	)
}

func TestContentMessage(t *testing.T) {
	t.Run(
		"content 消息", func(t *testing.T) {
			message, err := api.PostMessage(
				ctx, testChannelID, &dto.MessageToCreate{
					Content: "文本消息",
				},
			)
			if err != nil {
				t.Error(err)
			}
			t.Logf("message : %v", message)
		},
	)
}

func TestPatchMessage(t *testing.T) {
	t.Run(
		"修改消息", func(t *testing.T) {
			message, err := api.PatchMessage(
				ctx, testChannelID, testMessageID, &dto.MessageToCreate{
					Keyboard: &keyboard.MessageKeyboard{
						ID: "62",
					},
					Markdown: &dto.Markdown{
						TemplateID: 65,
						Params: []*dto.MarkdownParams{
							{
								Key:    "title",
								Values: []string{"标题"},
							},
							{
								Key:    "content",
								Values: []string{"内容"},
							},
						},
					},
				},
			)
			if err != nil {
				t.Error(err)
			}
			t.Logf("message : %v", message)
		},
	)
}
