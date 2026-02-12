# Rugo Compiler Fuzzer

Coverage-guided fuzzer using Go's native `testing.Fuzz` to find crashes and bad error messages in the compiler pipeline.

## Quick Start

Three fuzz targets live in `compiler/fuzz_test.go`:

```bash
# Fuzz the parser (preprocessor → parser → AST walker)
go test ./compiler/ -fuzz FuzzParseSource -fuzztime 120s

# Fuzz codegen (parse + code generation)
go test ./compiler/ -fuzz FuzzCodegen -fuzztime 120s

# Fuzz just the preprocessor pipeline
go test ./compiler/ -fuzz FuzzPreprocessor -fuzztime 120s
```

The corpus is seeded with all `.rugo` files from `examples/` and `rats/fixtures/` (~437 files) plus hand-crafted edge cases. Go's fuzzer mutates these at the byte level and uses coverage feedback to steer toward new code paths.

Failing inputs are saved to `compiler/testdata/fuzz/` automatically. Re-run a specific failure with:

```bash
go test ./compiler/ -run FuzzCodegen/<hash>
```

## Architecture

### Pipeline under test

```
Source → ExpandHeredocs → StripComments → ExpandStructDefs → Preprocess → Parser → Walker → Codegen
```

- **FuzzPreprocessor** — exercises the full pipeline (panics anywhere are caught by Go's test runner)
- **FuzzParseSource** — same pipeline, but also checks error messages for "internal compiler error" and "runtime error" patterns
- **FuzzCodegen** — parses first, then feeds valid ASTs to `generate()` to find codegen-specific crashes

### What counts as a bug

The fuzzer flags two things:

1. **Panics** — the compiler should never panic on user input, no matter how malformed
2. **Bad error messages** — errors containing `"internal compiler error"`, `"runtime error"`, `"index out of range"`, or `"nil pointer"` indicate a Go-level crash was caught but not translated into a user-friendly message

## Tips for Next Time

### What worked

- **Coverage-guided fuzzing found bugs that hand-crafted inputs missed.** The `ProcessInterpolation` panic (#006) was only discovered by `FuzzCodegen` because it requires a specific string that parses successfully but crashes during code generation.
- **Seeding with real files is critical.** The 437 `.rugo` files give the fuzzer a huge head start — it already knows valid syntax and mutates from there.
- **Fuzzing deeper pipeline stages pays off.** `FuzzParseSource` alone missed the codegen bugs because they only trigger on inputs that *parse successfully* but contain edge-case AST structures.

### What to try next

- **Fuzz `Compile`/`Emit` end-to-end** and check that the generated Go source actually compiles with `go build`. This catches codegen bugs that produce syntactically invalid Go.
- **Grammar-aware generation** — write a generator that walks the EBNF and produces random *syntactically valid* programs with extreme properties (huge arity, 1000-statement bodies, deeply nested expressions). Valid-but-extreme inputs stress the walker and codegen more than invalid ones do.
- **Fuzz with `TestMode: true`** to exercise the `rats` and `bench` codegen paths.
- **Fuzz `require` resolution** by creating temporary `.rugo` files on disk and fuzzing the multi-file compilation path.
- **Increase fuzz time.** 2 minutes found 2 new bugs on codegen. Running overnight with `-fuzztime 0` (unlimited) would likely find more.
- **Add new seeds when bugs are fixed.** After fixing a crash, add the triggering input as a seed in `seedCorpus()` so the fuzzer always re-tests it.

### Known skips in fuzz_test.go

The fuzz harness skips known bugs to keep finding new ones:

- `FuzzParseSource` skips `"invalid assignment target"` (issue #005)
- `FuzzCodegen` skips `"slice bounds out of range"` panics (issue #006) and `"interpolation error"` (issue #007)

**Remove these skips after fixing the corresponding bugs** so the fuzzer can verify the fixes and catch regressions.
