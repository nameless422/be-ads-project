package normalizer

import (
	"encoding/json"
	"strconv"
	"time"
)

func decode(recordPayload []byte, target any) error {
	return json.Unmarshal(recordPayload, target)
}

func parseRFC3339Ptr(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &value
}

func parseDate(raw string) time.Time {
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}
	}
	return value
}

func parseInt64(raw string) int64 {
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func microsToDecimalString(raw string) string {
	if raw == "" {
		return "0"
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "0"
	}
	return strconv.FormatFloat(value/1_000_000, 'f', 2, 64)
}
