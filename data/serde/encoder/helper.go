package encoder

import (
	"fmt"
	"strings"
)

const symbol = ">"

func wrapString(s string, limit int) string {
	if limit <= 0 {
		return s
	}

	var builder strings.Builder
	for i := 0; i < len(s); i += limit {
		end := i + limit
		if end > len(s) {
			end = len(s)
		}
		builder.WriteString(s[i:end])
		if end-i == limit && end != len(s) {
			builder.WriteString(fmt.Sprintf(" %s\n", symbol))
		}
	}
	return builder.String()
}

func unwrapString(s string) string {
	lines := strings.Split(s, "\n")
	var builder strings.Builder

	for _, line := range lines {
		line = strings.TrimSuffix(line, fmt.Sprintf(" %s", symbol))
		builder.WriteString(line)
	}

	return strings.TrimSpace(builder.String())
}
