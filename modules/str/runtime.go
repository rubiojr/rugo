package strmod

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// --- str module ---

type Str struct{}

func (*Str) Contains(s, substr string) interface{} {
	return strings.Contains(s, substr)
}

func (*Str) Split(s, sep string) interface{} {
	parts := strings.Split(s, sep)
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}

func (*Str) Trim(s string) interface{} {
	return strings.TrimSpace(s)
}

func (*Str) StartsWith(s, prefix string) interface{} {
	return strings.HasPrefix(s, prefix)
}

func (*Str) EndsWith(s, suffix string) interface{} {
	return strings.HasSuffix(s, suffix)
}

func (*Str) Replace(s, old, new string) interface{} {
	return strings.ReplaceAll(s, old, new)
}

func (*Str) Upper(s string) interface{} {
	return strings.ToUpper(s)
}

func (*Str) Lower(s string) interface{} {
	return strings.ToLower(s)
}

func (*Str) Index(s, substr string) interface{} {
	return strings.Index(s, substr)
}

func (*Str) Join(v interface{}, sep string) interface{} {
	parts, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("str.join() expects an array, got %T", v))
	}
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = fmt.Sprintf("%v", p)
	}
	return strings.Join(strs, sep)
}

func (*Str) RuneCount(s string) interface{} {
	return utf8.RuneCountInString(s)
}
