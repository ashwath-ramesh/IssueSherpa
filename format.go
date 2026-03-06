package main

import (
	"time"
)

func FormatDate(iso string) string {
	if iso == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, iso); err == nil {
		return parsed.Format("Jan 02")
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05", iso); err == nil {
		return parsed.Format("Jan 02")
	}
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}
