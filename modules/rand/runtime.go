package randmod

import (
	"fmt"
	"math/rand/v2"
)

// --- rand module ---

type Rand struct{}

func (*Rand) Int(min, max int) interface{} {
	if min >= max {
		panic(fmt.Sprintf("rand.int: min (%d) must be less than max (%d)", min, max))
	}
	return rand.IntN(max-min) + min
}

func (*Rand) Float() interface{} {
	return rand.Float64()
}

func (*Rand) String(length int) interface{} {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.IntN(len(chars))]
	}
	return string(b)
}

func (*Rand) Choice(arr interface{}) interface{} {
	items, ok := arr.([]interface{})
	if !ok {
		panic(fmt.Sprintf("rand.choice() expects an array, got %T", arr))
	}
	if len(items) == 0 {
		panic("rand.choice: array is empty")
	}
	return items[rand.IntN(len(items))]
}

func (*Rand) Shuffle(arr interface{}) interface{} {
	items, ok := arr.([]interface{})
	if !ok {
		panic(fmt.Sprintf("rand.shuffle() expects an array, got %T", arr))
	}
	result := make([]interface{}, len(items))
	copy(result, items)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

func (*Rand) Uuid() interface{} {
	var uuid [16]byte
	for i := range uuid {
		uuid[i] = byte(rand.IntN(256))
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
