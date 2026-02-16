package timemod

import (
	"fmt"
	"time"
)

// --- time module ---

type Time struct{}

func (*Time) Now() interface{} {
	return float64(time.Now().UnixNano()) / 1e9
}

func (*Time) Sleep(seconds float64) interface{} {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
	return nil
}

func (*Time) Format(timestamp float64, layout string) interface{} {
	sec := int64(timestamp)
	nsec := int64((timestamp - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec)
	return t.Format(layout)
}

func (*Time) Parse(s, layout string) interface{} {
	t, err := time.Parse(layout, s)
	if err != nil {
		panic(fmt.Sprintf("time.parse failed: %v", err))
	}
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}

func (*Time) Since(timestamp float64) interface{} {
	return float64(time.Now().UnixNano())/1e9 - timestamp
}

func (*Time) Millis() interface{} {
	return int(time.Now().UnixMilli())
}
