.PHONY: build test rats baseline baseline-clean rats-compare bench-compare stats

build:
	@go build -o bin/rugo .

test:
	@go test ./... -count=1

rats: build
	@PATH="$(CURDIR)/bin:$(PATH)" bin/rugo rats --recap --timing rats/

# --- Baseline workflow -------------------------------------------------------
#
# `make baseline` snapshots the current binary as bin/rugo-baseline. Re-running
# it overwrites the snapshot, so checkout an older revision first when you
# want to capture a "before" build.
#
# `make rats-compare` runs the entire RATS suite twice (baseline first, then
# current bin/rugo) and reports total wall-clock time for each so you can see
# the impact of compiler changes on real test runs.
#
# `make bench-compare` runs the type-inference benchmarks with inference on
# (current default) and off (--no-infer), so the inference engine's payoff is
# visible in numbers.

baseline: build
	@cp bin/rugo bin/rugo-baseline
	@echo "Saved baseline binary: bin/rugo-baseline"
	@bin/rugo-baseline --version 2>/dev/null || true

baseline-clean:
	@rm -f bin/rugo-baseline

rats-compare: build
	@if [ ! -x bin/rugo-baseline ]; then \
	  echo "error: bin/rugo-baseline missing — run 'make baseline' on the old revision first"; \
	  exit 1; \
	fi
	@echo "=== baseline ($$(bin/rugo-baseline --version 2>/dev/null || echo unknown)) ==="
	@start=$$(date +%s); \
	PATH="$(CURDIR)/bin:$(PATH)" bin/rugo-baseline rats --recap rats/ > /tmp/rats-baseline.log 2>&1 || true; \
	end=$$(date +%s); echo "baseline: $$((end-start))s wall"; tail -3 /tmp/rats-baseline.log
	@echo
	@echo "=== current ($$(bin/rugo --version 2>/dev/null || echo unknown)) ==="
	@start=$$(date +%s); \
	PATH="$(CURDIR)/bin:$(PATH)" bin/rugo rats --recap rats/ > /tmp/rats-current.log 2>&1 || true; \
	end=$$(date +%s); echo "current : $$((end-start))s wall"; tail -3 /tmp/rats-current.log
	@echo
	@echo "Full logs: /tmp/rats-baseline.log /tmp/rats-current.log"

bench-compare: build
	@if [ ! -d bench ]; then echo "no bench/ directory"; exit 1; fi
	@echo "=== inferred (default) ==="
	@PATH="$(CURDIR)/bin:$(PATH)" bin/rugo bench bench/
	@echo
	@echo "=== --no-infer ==="
	@PATH="$(CURDIR)/bin:$(PATH)" bin/rugo bench --no-infer bench/

# Type coverage stats: helpful for tracking inference improvements over time.
# Usage: make stats FILE=path/to/script.rugo
stats: build
	@if [ -z "$(FILE)" ]; then echo "usage: make stats FILE=path/to/script.rugo"; exit 1; fi
	@bin/rugo emit --stats $(FILE)

