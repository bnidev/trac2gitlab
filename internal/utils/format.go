package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func TracToMarkdown(input string) string {
	// Multiline code blocks: {{{#!lang\n...\n}}} or just {{{\n...\n}}}
	reCodeBlock := regexp.MustCompile(`(?s){{{\s*(?:#!(\w+))?\s*\n(.*?)\n\s*}}}`)
	input = reCodeBlock.ReplaceAllStringFunc(input, func(m string) string {
		matches := reCodeBlock.FindStringSubmatch(m)
		if len(matches) < 3 {
			return m // fallback
		}
		lang := matches[1]
		code := matches[2]
		if lang != "" {
			return "```" + lang + "\n" + code + "\n```"
		}
		return "```\n" + code + "\n```"
	})

	// Inline code: {{{code}}} (not multiline)
	reInlineCode := regexp.MustCompile(`{{{([^}\n]+)}}}`)
	input = reInlineCode.ReplaceAllString(input, "`$1`")

	// Headings: = Heading 1, == Heading 2, etc.
	reHeading := regexp.MustCompile(`(?m)^(={1,6})\s*(.+)$`)
	input = reHeading.ReplaceAllStringFunc(input, func(m string) string {
		parts := reHeading.FindStringSubmatch(m)
		level := len(parts[1])
		return strings.Repeat("#", level) + " " + parts[2]
	})

	// Font styles
	input = regexp.MustCompile(`'''''(.*?)'''''`).ReplaceAllString(input, `***$1***`) // bold+italic
	input = regexp.MustCompile(`'''(.*?)'''`).ReplaceAllString(input, `**$1**`)       // bold
	input = regexp.MustCompile(`''(.*?)''`).ReplaceAllString(input, `*$1*`)           // italic

	// underline, subscript, superscript
	input = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(input, `<u>$1</u>`)
	input = regexp.MustCompile(`\^([^^]+)\^`).ReplaceAllString(input, `<sup>$1</sup>`)
	input = regexp.MustCompile(`,,([^,]+),,`).ReplaceAllString(input, `<sub>$1</sub>`)

	// Line breaks
	input = strings.ReplaceAll(input, "[[BR]]", "  \n")
	input = strings.ReplaceAll(input, "[[br]]", "  \n")

	// Horizontal rule
	input = regexp.MustCompile(`(?m)^----\s*$`).ReplaceAllString(input, `---`)

	// External links: [http://url Label]
	input = regexp.MustCompile(`\[(https?://[^\s\]]+)\s+([^\]]+)\]`).ReplaceAllString(input, `[$2]($1)`)

	// Wiki links: [wiki:Page Label] → [Label](Page.md)
	input = regexp.MustCompile(`\[wiki:([^\s\]]+)\s+([^\]]+)\]`).ReplaceAllString(input, `[$2]($1.md)`)

	// Unordered lists: * item or - item
	input = regexp.MustCompile(`(?m)^ *[\*\-] +(.*)$`).ReplaceAllString(input, "* $1")

	// Ordered lists: 1. item, a. item, i. item
	input = regexp.MustCompile(`(?m)^ *\d+\.\s+(.*)$`).ReplaceAllString(input, "1. $1")

	// Definition lists: term:: definition
	input = regexp.MustCompile(`(?m)^(.+?)::\s*(.+)$`).ReplaceAllString(input, "$1\n: $2")

	// Blockquotes: > or indented
	input = regexp.MustCompile(`(?m)^ {2,}(.+)$`).ReplaceAllString(input, "> $1")

	return input
}

func ParseTracTime(val any) (*time.Time, error) {
	switch t := val.(type) {
	case int64:
		if t == 0 {
			return nil, nil
		}
		parsed := time.Unix(t, 0).UTC()
		return &parsed, nil
	case int:
		if t == 0 {
			return nil, nil
		}
		parsed := time.Unix(int64(t), 0).UTC()
		return &parsed, nil
	case string:
		if t == "" {
			return nil, nil
		}
		parsed, err := time.Parse("2006-01-02T15:04:05", t)
		if err != nil {
			return nil, err
		}
		return &parsed, nil
	case time.Time:
		return &t, nil
	default:
		return nil, fmt.Errorf("unknown time format: %T", t)
	}
}

func ParseRequiredTracTime(val any) (time.Time, error) {
	t, err := ParseTracTime(val)
	if err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, fmt.Errorf("required time value is missing or zero")
	}
	return *t, nil
}

func ToInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int32:
		return int64(t), true
	case float64:
		return int64(t), true
	case int:
		return int64(t), true
	default:
		return 0, false
	}
}
