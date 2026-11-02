package dto

import (
	"encoding/json"
	"strconv"
	"time"
)

// Timestamp supports legacy RFC3339 guild timestamps and QQ v2 Unix seconds.
type Timestamp string

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = ""
		return nil
	}
	var text string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
	} else {
		var seconds json.Number
		if err := json.Unmarshal(data, &seconds); err != nil {
			return err
		}
		if _, err := seconds.Int64(); err != nil {
			return err
		}
		text = seconds.String()
	}
	*t = Timestamp(text)
	return nil
}
func (t Timestamp) Time() (time.Time, error) {
	if seconds, err := strconv.ParseInt(string(t), 10, 64); err == nil {
		return time.Unix(seconds, 0), nil
	}
	return time.Parse(time.RFC3339, string(t))
}
