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

func (*Str) Count(s, substr string) interface{} {
	return strings.Count(s, substr)
}

func (*Str) Repeat(s string, n int) interface{} {
	return strings.Repeat(s, n)
}

func (*Str) Reverse(s string) interface{} {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func (*Str) Chars(s string) interface{} {
	runes := []rune(s)
	result := make([]interface{}, len(runes))
	for i, r := range runes {
		result[i] = string(r)
	}
	return result
}

func (*Str) Fields(s string) interface{} {
	parts := strings.Fields(s)
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}

func (*Str) TrimPrefix(s, prefix string) interface{} {
	return strings.TrimPrefix(s, prefix)
}

func (*Str) TrimSuffix(s, suffix string) interface{} {
	return strings.TrimSuffix(s, suffix)
}

func (*Str) PadLeft(s string, width int, extra ...interface{}) interface{} {
	pad := " "
	if len(extra) > 0 {
		pad = fmt.Sprintf("%v", extra[0])
	}
	rc := utf8.RuneCountInString(s)
	if rc >= width {
		return s
	}
	needed := width - rc
	var b strings.Builder
	padRunes := []rune(pad)
	for i := 0; i < needed; i++ {
		b.WriteRune(padRunes[i%len(padRunes)])
	}
	b.WriteString(s)
	return b.String()
}

func (*Str) PadRight(s string, width int, extra ...interface{}) interface{} {
	pad := " "
	if len(extra) > 0 {
		pad = fmt.Sprintf("%v", extra[0])
	}
	rc := utf8.RuneCountInString(s)
	if rc >= width {
		return s
	}
	needed := width - rc
	var b strings.Builder
	b.WriteString(s)
	padRunes := []rune(pad)
	for i := 0; i < needed; i++ {
		b.WriteRune(padRunes[i%len(padRunes)])
	}
	return b.String()
}

func (*Str) EachLine(s string) interface{} {
	lines := strings.Split(s, "\n")
	result := make([]interface{}, len(lines))
	for i, l := range lines {
		result[i] = l
	}
	return result
}

func (*Str) Center(s string, width int, extra ...interface{}) interface{} {
	pad := " "
	if len(extra) > 0 {
		pad = fmt.Sprintf("%v", extra[0])
	}
	rc := utf8.RuneCountInString(s)
	if rc >= width {
		return s
	}
	total := width - rc
	left := total / 2
	right := total - left
	var b strings.Builder
	padRunes := []rune(pad)
	for i := 0; i < left; i++ {
		b.WriteRune(padRunes[i%len(padRunes)])
	}
	b.WriteString(s)
	for i := 0; i < right; i++ {
		b.WriteRune(padRunes[i%len(padRunes)])
	}
	return b.String()
}

func (*Str) LastIndex(s, substr string) interface{} {
	return strings.LastIndex(s, substr)
}

func (*Str) Slice(s string, start, end int) interface{} {
	runes := []rune(s)
	n := len(runes)
	if start < 0 {
		start += n
	}
	if end < 0 {
		end += n
	}
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if start > n || start >= end {
		return ""
	}
	return string(runes[start:end])
}

func (*Str) Empty(s string) interface{} {
	return s == ""
}

func (*Str) ByteSize(s string) interface{} {
	return len(s)
}
