package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func TracToMarkdown(input string) string {
	// Multiline code blocks: {{{#!lang\n...\n}}}, closed or unclosed.
	input = convertCodeBlocks(input)

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

	// Wiki links: [wiki:Page Label] / [wiki:Page] → [Label](Page.md)
	reWikiLink := regexp.MustCompile(`\[wiki:([^\s\]]+)(?:\s+([^\]]+))?\]`)
	input = reWikiLink.ReplaceAllStringFunc(input, func(m string) string {
		parts := reWikiLink.FindStringSubmatch(m)
		page, label := parts[1], parts[2]
		if label == "" {
			label = page
		}
		return "[" + label + "](" + page + ".md)"
	})

	// Trac inline links/anchors: [[Page]], [[Page|label]], [[#anchor|label]].
	// Macros like [[Image(...)]] are left for the importer (they need URLs).
	input = convertTracLinks(input)

	// Unordered lists: * item or - item
	input = regexp.MustCompile(`(?m)^ *[\*\-] +(.*)$`).ReplaceAllString(input, "* $1")

	// Ordered lists: 1. item, a. item, i. item
	input = regexp.MustCompile(`(?m)^ *\d+\.\s+(.*)$`).ReplaceAllString(input, "1. $1")

	// Definition lists: term:: definition
	input = regexp.MustCompile(`(?m)^(.+?)::\s*(.+)$`).ReplaceAllString(input, "$1\n: $2")

	// Blockquotes: > or indented
	input = regexp.MustCompile(`(?m)^ {2,}(.+)$`).ReplaceAllString(input, "> $1")

	// Tables: ||cell||cell|| (after inline conversions so cells are markdown)
	input = convertTables(input)

	return input
}

var (
	reCodeClosed = regexp.MustCompile(`(?s){{{\s*(?:#!(\w+))?\s*\n(.*?)\n\s*}}}`)
	reCodeOpen   = regexp.MustCompile(`(?s){{{\s*(?:#!(\w+))?\s*\n(.*)$`)
	reTracLink   = regexp.MustCompile(`\[\[([^\]()|]+)(?:\|([^\]]+))?\]\]`)
	reTableRow   = regexp.MustCompile(`^\s*\|\|.*\|\|\s*$`)
	reTracEscape = regexp.MustCompile(`(^|\s)!(\w)`)
)

// convertCodeBlocks turns Trac {{{ ... }}} blocks into fenced code. Trac emits
// code to the end of the page when the block is never closed, so the unclosed
// form is handled too.
func convertCodeBlocks(input string) string {
	input = reCodeClosed.ReplaceAllStringFunc(input, func(m string) string {
		parts := reCodeClosed.FindStringSubmatch(m)
		return codeFence(parts[1], parts[2])
	})
	return reCodeOpen.ReplaceAllStringFunc(input, func(m string) string {
		parts := reCodeOpen.FindStringSubmatch(m)
		return codeFence(parts[1], parts[2])
	})
}

func codeFence(lang, code string) string {
	if lang != "" {
		return "```" + lang + "\n" + code + "\n```"
	}
	return "```\n" + code + "\n```"
}

// convertTracLinks rewrites [[Page]], [[Page|label]] and [[#anchor|label]].
// The character class excludes parentheses, so [[Macro(...)]] is left alone.
func convertTracLinks(input string) string {
	return reTracLink.ReplaceAllStringFunc(input, func(m string) string {
		parts := reTracLink.FindStringSubmatch(m)
		target := strings.TrimSpace(parts[1])
		label := strings.TrimSpace(parts[2])
		if label == "" {
			label = target
		}
		if strings.HasPrefix(target, "#") {
			return "[" + label + "](" + target + ")"
		}
		return "[" + label + "](" + target + ".md)"
	})
}

type tableCell struct {
	text   string
	header bool
}

// convertTables rewrites runs of ||...|| rows as GitHub-flavoured tables. Lines
// inside a fenced code block are left untouched.
func convertTables(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	inFence := false
	for i := 0; i < len(lines); {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			out = append(out, lines[i])
			i++
			continue
		}
		if inFence || !reTableRow.MatchString(lines[i]) {
			out = append(out, lines[i])
			i++
			continue
		}
		j := i
		for j < len(lines) && reTableRow.MatchString(lines[j]) {
			j++
		}
		out = append(out, renderTable(lines[i:j])...)
		i = j
	}
	return strings.Join(out, "\n")
}

func renderTable(rows []string) []string {
	parsed := make([][]tableCell, 0, len(rows))
	cols := 0
	for _, r := range rows {
		cells := parseTableRow(r)
		parsed = append(parsed, cells)
		if len(cells) > cols {
			cols = len(cells)
		}
	}
	out := make([]string, 0, len(parsed)+1)
	for ri, cells := range parsed {
		var b strings.Builder
		b.WriteString("|")
		for c := 0; c < cols; c++ {
			text, header := "", false
			if c < len(cells) {
				text, header = cells[c].text, cells[c].header
			}
			text = strings.ReplaceAll(text, "|", `\|`)
			if header && ri > 0 {
				text = "**" + text + "**"
			}
			b.WriteString(" " + text + " |")
		}
		out = append(out, b.String())
		if ri == 0 {
			var s strings.Builder
			s.WriteString("|")
			for c := 0; c < cols; c++ {
				s.WriteString(" --- |")
			}
			out = append(out, s.String())
		}
	}
	return out
}

func parseTableRow(line string) []tableCell {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "||")
	line = strings.TrimSuffix(line, "||")
	parts := strings.Split(line, "||")
	cells := make([]tableCell, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		header := false
		if strings.HasPrefix(p, "=") {
			header = true
			p = strings.TrimSpace(strings.Trim(p, "="))
		}
		p = reTracEscape.ReplaceAllString(p, "$1$2")
		cells = append(cells, tableCell{text: p, header: header})
	}
	return cells
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

func GetString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func GetInt64(v any) int64 {
	switch i := v.(type) {
	case int64:
		return i
	case int:
		return int64(i)
	case float64:
		return int64(i)
	default:
		return 0
	}
}

func GetBool(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func GetInt(v any) int {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

func GetStringSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
