// Package parser provides structured log parsing capabilities.
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Geun-Oh/lx/internal/entry"
)

// builtinPatterns provides commonly used Grok-style named patterns.
var builtinPatterns = map[string]string{
	"IP":         `(?:\d{1,3}\.){3}\d{1,3}`,
	"IPV6":       `[0-9A-Fa-f:]+`,
	"WORD":       `\w+`,
	"INT":        `[+-]?\d+`,
	"NUMBER":     `[+-]?(?:\d+\.?\d*|\.\d+)`,
	"NOTSPACE":   `\S+`,
	"DATA":       `.*?`,
	"GREEDYDATA": `.*`,
	"TIMESTAMP":  `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`,
	"LOGLEVEL":   `(?:DEBUG|INFO|WARN(?:ING)?|ERROR|ERR|FATAL|PANIC|CRITICAL|TRACE)`,
	"PATH":       `(?:/[\w.]+)+`,
	"URI":        `\S+://\S+`,
	"UUID":       `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	"MAC":        `(?:[0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}`,
	"HTTPMETHOD": `(?:GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|CONNECT|TRACE)`,
	"STATUSCODE": `\d{3}`,
	"QS":         `"[^"]*"`,
}

// GrokParser parses unstructured log lines using Grok-style patterns.
// Pattern format: %{PATTERN_NAME:capture_name}
// Example: "%{IP:client} %{WORD:method} %{NOTSPACE:path} %{STATUSCODE:status}"
type GrokParser struct {
	pattern    string
	regex      *regexp.Regexp
	fieldNames []string
}

// NewGrokParser compiles a Grok pattern string into a regex-based parser.
func NewGrokParser(pattern string) (*GrokParser, error) {
	regexStr, fieldNames, err := compileGrokPattern(pattern)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, fmt.Errorf("compiled grok regex invalid: %w (regex: %s)", err, regexStr)
	}

	return &GrokParser{
		pattern:    pattern,
		regex:      re,
		fieldNames: fieldNames,
	}, nil
}

// Parse extracts structured fields from a log entry's message.
// Returns true if the pattern matched and fields were extracted.
func (g *GrokParser) Parse(e *entry.LogEntry) bool {
	matches := g.regex.FindStringSubmatch(e.Message)
	if matches == nil {
		return false
	}

	if e.Fields == nil {
		e.Fields = make(map[string]string, len(g.fieldNames))
	}

	for i, name := range g.fieldNames {
		if i+1 < len(matches) && name != "" {
			e.Fields[name] = matches[i+1]
		}
	}

	return true
}

// Pattern returns the original Grok pattern string.
func (g *GrokParser) Pattern() string {
	return g.pattern
}

// compileGrokPattern converts a Grok pattern to a Go regex with named groups.
// %{PATTERN_NAME:field_name} → (?P<field_name>regex_for_PATTERN_NAME)
// %{PATTERN_NAME} → (?:regex_for_PATTERN_NAME)
func compileGrokPattern(pattern string) (string, []string, error) {
	var fieldNames []string
	result := pattern

	// Find all %{...} tokens.
	grokRe := regexp.MustCompile(`%\{(\w+)(?::(\w+))?\}`)
	matches := grokRe.FindAllStringSubmatch(pattern, -1)

	for _, m := range matches {
		fullMatch := m[0]
		patternName := m[1]
		fieldName := ""
		if len(m) > 2 {
			fieldName = m[2]
		}

		builtinRegex, ok := builtinPatterns[patternName]
		if !ok {
			return "", nil, fmt.Errorf("unknown grok pattern: %s", patternName)
		}

		var replacement string
		if fieldName != "" {
			replacement = fmt.Sprintf("(%s)", builtinRegex)
			fieldNames = append(fieldNames, fieldName)
		} else {
			replacement = fmt.Sprintf("(?:%s)", builtinRegex)
			fieldNames = append(fieldNames, "")
		}

		result = strings.Replace(result, fullMatch, replacement, 1)
	}

	return result, fieldNames, nil
}
