package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// AllDigits returns true if all digits of string is digit
func AllDigits(s string) bool {
	isNotDigit := func(c rune) bool { return c < '0' || c > '9' }
	return strings.IndexFunc(s, isNotDigit) == -1
}

// PadString pad string with single character to given length
func PadString(s string, p string, n int, left bool) string {
	if len(s) >= n {
		return s
	}
	if left {
		return fmt.Sprintf("%s%s", strings.Repeat(p, n-len(s)), s)
	}
	return fmt.Sprintf("%s%s", s, strings.Repeat(p, n-len(s)))
}

// Trim trim common characters
func Trim(s string) string {
	return strings.Trim(s, "\t\n\r\x00\x0B")
}

// UnescapeUnicode un-escapes unicode string
func UnescapeUnicode(s string) (string, error) {
	s, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(s), `\\u`, `\u`))
	if err != nil {
		return "", err
	}
	return s, nil
}
