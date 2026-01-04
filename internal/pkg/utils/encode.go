package utils

import (
	"html"
	"strings"
)

var (
	unicodeAltBase = strings.NewReplacer(
		"&", "＆",
		"<", "＜",
		">", "＞",
	)

	unicodeAltExtended = strings.NewReplacer(
		"&", "＆",
		"<", "＜",
		">", "＞",
		"'", "’",
		`"`, "＂",
		"^", "＾",
		"?", "？",
		`\`, "＼",
		"/", "／",
		",", "，",
		";", "；",
	)
)

// EscapeToUnicodeAlternative escapes ASCII characters into Unicode lookalikes
func EscapeToUnicodeAlternative(text string, extended bool) string {
	text = html.UnescapeString(text)
	if extended {
		return unicodeAltExtended.Replace(text)
	}
	return unicodeAltBase.Replace(text)
}
