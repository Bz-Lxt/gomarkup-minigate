package timeutil

import (
	"time"
)

var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func Parse(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, Beijing)
}
