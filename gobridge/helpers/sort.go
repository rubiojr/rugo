//go:build ignore

package helpers

import "sort"

func rugo_sort_in_place(v interface{}) {
	arr, ok := v.([]interface{})
	if !ok {
		panic("expected array for sort")
	}
	sort.Slice(arr, func(i, j int) bool {
		return rugo_compare(arr[i], arr[j]) < 0
	})
}

func rugo_sort_is_sorted(v interface{}) bool {
	arr, ok := v.([]interface{})
	if !ok {
		panic("expected array for sort")
	}
	return sort.SliceIsSorted(arr, func(i, j int) bool {
		return rugo_compare(arr[i], arr[j]) < 0
	})
}
