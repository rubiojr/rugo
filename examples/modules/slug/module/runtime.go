//go:build ignore

package slug

import (
	"strings"

	gosimpleslug "github.com/gosimple/slug"
)

type Slug struct{}

// Make generates a URL-friendly slug from text.
func (*Slug) Make(text string) interface{} {
	return gosimpleslug.Make(text)
}

// MakeLang generates a slug using language-specific transliteration.
func (*Slug) MakeLang(text string, lang string) interface{} {
	return gosimpleslug.MakeLang(text, lang)
}

// IsSlug checks if a string is already a valid slug.
func (*Slug) IsSlug(text string) interface{} {
	return gosimpleslug.IsSlug(text)
}

// Join combines multiple strings into a slug separated by hyphens.
func (*Slug) Join(parts ...interface{}) interface{} {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = rugo_to_string(p)
	}
	return gosimpleslug.Make(strings.Join(strs, " "))
}
