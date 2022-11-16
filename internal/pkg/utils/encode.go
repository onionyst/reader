package utils

import (
	"html"
	"strings"
)

// EscapeToUnicodeAlternative escape characters to UNICODE alternatives
func EscapeToUnicodeAlternative(text string, extended bool) string {
	text = html.UnescapeString(text)

	problem := []string{"&", "<", ">"}
	replace := []string{"＆", "＜", "＞"}

	if extended {
		problem = append(problem, "'", "\"", "^", "?", "\\", "/", ",", ";")
		replace = append(replace, "’", "＂", "＾", "？", "＼", "／", "，", "；")
	}

	for i := range problem {
		text = strings.ReplaceAll(text, problem[i], replace[i])
	}

	return text
}
