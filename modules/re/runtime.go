//go:build ignore

package remod

import (
	"fmt"
	"regexp"
)

// --- re module ---

type Re struct{}

// Test returns true if the pattern matches the string.
func (*Re) Test(pattern, s string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.test: invalid pattern %q: %v", pattern, err))
	}
	return re.MatchString(s)
}

// Find returns the first match, or nil if no match.
func (*Re) Find(pattern, s string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.find: invalid pattern %q: %v", pattern, err))
	}
	match := re.FindString(s)
	if match == "" {
		return nil
	}
	return match
}

// FindAll returns all matches as an array.
func (*Re) FindAll(pattern, s string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.find_all: invalid pattern %q: %v", pattern, err))
	}
	matches := re.FindAllString(s, -1)
	result := make([]interface{}, len(matches))
	for i, m := range matches {
		result[i] = m
	}
	return result
}

// Replace replaces the first match with the replacement string.
func (*Re) Replace(pattern, s, repl string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.replace: invalid pattern %q: %v", pattern, err))
	}
	loc := re.FindStringIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[0]] + repl + s[loc[1]:]
}

// ReplaceAll replaces all matches with the replacement string.
func (*Re) ReplaceAll(pattern, s, repl string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.replace_all: invalid pattern %q: %v", pattern, err))
	}
	return re.ReplaceAllString(s, repl)
}

// Split splits the string by the pattern.
func (*Re) Split(pattern, s string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.split: invalid pattern %q: %v", pattern, err))
	}
	parts := re.Split(s, -1)
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}

// Match returns a hash with "match" (full match) and "groups" (capture groups),
// or nil if no match.
func (*Re) Match(pattern, s string) interface{} {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("re.match: invalid pattern %q: %v", pattern, err))
	}
	submatch := re.FindStringSubmatch(s)
	if submatch == nil {
		return nil
	}
	groups := make([]interface{}, 0, len(submatch)-1)
	for _, g := range submatch[1:] {
		groups = append(groups, g)
	}
	result := map[interface{}]interface{}{
		"match":  submatch[0],
		"groups": groups,
	}
	return result
}
