//go:build ignore

package helpers

func rugo_first_rune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}
