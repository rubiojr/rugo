//go:build ignore

package benchmod

import (
	"fmt"
	"os"
	"time"
)

// --- bench module ---

type Bench struct{}

// rugoBenchCase describes a single benchmark for the runner.
type rugoBenchCase struct {
	Name string
	Func func()
}

// rugo_bench_runner executes benchmarks with auto-calibration and reports timing.
func rugo_bench_runner(benches []rugoBenchCase) {
	if len(benches) == 0 {
		fmt.Fprintln(os.Stderr, "no benchmarks to run")
		return
	}

	noColor := os.Getenv("NO_COLOR") != ""
	colorName := "\033[1m"
	colorTime := "\033[36m"
	colorRuns := "\033[33m"
	colorReset := "\033[0m"
	if noColor {
		colorName = ""
		colorTime = ""
		colorRuns = ""
		colorReset = ""
	}

	for _, b := range benches {
		// Warm up: run once to avoid cold-start effects
		b.Func()

		// Calibrate: find N such that total time is >= 1 second
		n := 1
		var elapsed time.Duration
		for {
			start := time.Now()
			for range n {
				b.Func()
			}
			elapsed = time.Since(start)
			if elapsed >= time.Second {
				break
			}
			// Scale up: aim for 1s based on current rate
			if elapsed > 0 {
				next := int(float64(n) * float64(time.Second) / float64(elapsed))
				if next <= n {
					next = n * 2
				}
				n = next
			} else {
				n *= 10
			}
		}

		nsPerOp := float64(elapsed.Nanoseconds()) / float64(n)
		timeStr := formatDuration(nsPerOp)
		fmt.Fprintf(os.Stderr, "  %s%-40s%s %s%10s/op%s %s(%d runs)%s\n",
			colorName, b.Name, colorReset,
			colorTime, timeStr, colorReset,
			colorRuns, n, colorReset)
	}
}

// formatDuration formats nanoseconds per operation in a human-readable way.
func formatDuration(ns float64) string {
	switch {
	case ns < 1000:
		return fmt.Sprintf("%.1f ns", ns)
	case ns < 1000000:
		return fmt.Sprintf("%.1f µs", ns/1000)
	case ns < 1000000000:
		return fmt.Sprintf("%.1f ms", ns/1000000)
	default:
		return fmt.Sprintf("%.2f s", ns/1000000000)
	}
}
