package config

import (
	"fmt"
	"strconv"
	"strings"
)

// parse turns a YAML-subset document into a tree of map[string]any, []any and
// scalar values (bool, int, string). It supports only what JobRadar's config
// needs: nested mappings, block sequences of scalars, "# comments", and 2-space
// indentation. Flow style ({}, []), anchors, multi-line strings, etc. are not
// supported by design.
func parse(src string) (any, error) {
	lines, err := tokenize(src)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return map[string]any{}, nil
	}
	val, _, err := parseBlock(lines, 0, lines[0].indent)
	return val, err
}

type line struct {
	indent int
	text   string
	num    int
}

func tokenize(src string) ([]line, error) {
	var out []line
	for i, raw := range strings.Split(src, "\n") {
		// Strip comments. Config values never contain '#', so a naive cut is safe.
		if idx := strings.IndexByte(raw, '#'); idx >= 0 {
			raw = raw[:idx]
		}
		if strings.TrimSpace(raw) == "" {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if strings.Contains(raw[:indent], "\t") {
			return nil, fmt.Errorf("line %d: tabs are not allowed for indentation", i+1)
		}
		out = append(out, line{indent: indent, text: strings.TrimRight(raw[indent:], " "), num: i + 1})
	}
	return out, nil
}

// parseBlock parses consecutive lines at the given indent and returns the value
// plus the index of the first line it did not consume.
func parseBlock(lines []line, i, indent int) (any, int, error) {
	if strings.HasPrefix(lines[i].text, "- ") || lines[i].text == "-" {
		return parseSequence(lines, i, indent)
	}
	return parseMapping(lines, i, indent)
}

func parseSequence(lines []line, i, indent int) (any, int, error) {
	seq := []any{}
	for i < len(lines) && lines[i].indent == indent && (strings.HasPrefix(lines[i].text, "- ") || lines[i].text == "-") {
		item := strings.TrimSpace(strings.TrimPrefix(lines[i].text, "-"))
		seq = append(seq, scalar(item))
		i++
	}
	return seq, i, nil
}

func parseMapping(lines []line, i, indent int) (any, int, error) {
	m := map[string]any{}
	for i < len(lines) && lines[i].indent == indent {
		key, val, hasInline := splitKV(lines[i].text)
		if key == "" {
			return nil, i, fmt.Errorf("line %d: expected \"key: value\", got %q", lines[i].num, lines[i].text)
		}
		i++
		switch {
		case hasInline:
			m[key] = scalar(val)
		case i < len(lines) && lines[i].indent > indent:
			child, ni, err := parseBlock(lines, i, lines[i].indent)
			if err != nil {
				return nil, ni, err
			}
			m[key] = child
			i = ni
		default:
			m[key] = nil // key with no value and no nested block
		}
	}
	return m, i, nil
}

// splitKV splits "key: value" into its parts. It returns hasInline=false for a
// bare "key:" that introduces a nested block.
func splitKV(text string) (key, val string, hasInline bool) {
	idx := strings.IndexByte(text, ':')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(text[:idx])
	rest := strings.TrimSpace(text[idx+1:])
	return key, rest, rest != ""
}

// scalar coerces a raw token into bool, int or string.
func scalar(s string) any {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	switch strings.ToLower(s) {
	case "true", "yes":
		return true
	case "false", "no":
		return false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return s
}
