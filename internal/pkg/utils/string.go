package utils

import (
	"strconv"
	"strings"
)

// AllDigits reports string non-empty and consists only of ASCII digits
func AllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// PadString pad string with single character to reach length
func PadString(s string, pad byte, n int, left bool) string {
	if pad == 0 || n <= 0 || len(s) >= n {
		return s
	}
	padding := strings.Repeat(string(pad), n-len(s))
	if left {
		return padding + s
	}
	return s + padding
}

// UnescapeUnicode un-escapes unicode string
func UnescapeUnicode(s string) (string, error) {
	s, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(s), `\\u`, `\u`))
	if err != nil {
		return "", err
	}
	return s, nil
}
