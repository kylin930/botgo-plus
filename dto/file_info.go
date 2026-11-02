package dto

import (
	"encoding/base64"
	"encoding/json"
)

// QQ file_info is opaque, not necessarily valid base64. Preserve it byte-for-byte
// while retaining the legacy []byte fields for existing SDK callers.
func (m *Message) UnmarshalJSON(data []byte) error {
	type alias Message
	var wire struct {
		*alias
		FileInfo string `json:"file_info"`
	}
	wire.alias = (*alias)(m)
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	m.FileInfoRaw = wire.FileInfo
	if wire.FileInfo != "" {
		decoded, err := base64.StdEncoding.DecodeString(wire.FileInfo)
		if err != nil {
			m.FileInfo = []byte(wire.FileInfo)
		} else {
			m.FileInfo = decoded
		}
	}
	return nil
}
func (m MediaInfo) MarshalJSON() ([]byte, error) {
	type alias MediaInfo
	if m.FileInfoRaw == "" {
		return json.Marshal(alias(m))
	}
	return json.Marshal(struct {
		alias
		FileInfo string `json:"file_info"`
	}{alias(m), m.FileInfoRaw})
}
