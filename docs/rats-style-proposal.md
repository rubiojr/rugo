# RATS as exemplar Rugo

Tracking: kata `eayc` — complete suite/fixture checklist and per-file evidence.

## Summary

Make a test read as a small example of the behavior it proves. Keep the input,
operation, and expected result together. Use the book's clear names, hashes,
closures, raw heredocs, and direct iteration. Spend abstraction on failure
messages and resource ownership.

This is a proposal, backed by runnable prototypes. It is not a suite-wide style
migration or a new assertion framework.

## Findings from the initial survey

These findings describe baseline `f3459f7`. The variables and array-method
pilots have since been refactored and fault-checked; kata `eayc` records their
completion evidence and the remaining work.

The suite has 201 test files and 2,628 tests. A textual survey of `_test.rugo`
files finds 1,082 `test.run` calls and 890 `eval.run`/`eval.file` calls, including
test fixtures. Many are necessary. Others turn a simple value assertion into a
second compilation and a search for an output marker.

Specific starting points:

- `rats/core/02_variables_test.rugo`: the test named "string interpolation"
  concatenates strings. Its body doesn't exercise its named feature.
- `rats/core/82_array_methods_test.rugo`: the `.each()` test runs a no-op
  callback and checks the input length. The map test checks individual indexes
  without checking whether there are unexpected extra results.
- `rats/core/90_collection_equality_test.rugo`: ten tests each execute the same
  large program and search for different success markers. Some marker strings
  contain other markers, making the checks less specific than they appear.
- `rats/core/collection_callback_args_test.rugo`: the recent callback tests
  compile 36 tiny programs for missing-argument behavior. Most of those checks
  can call the methods directly.
- `rats/stdlib/sqlite_test.rugo`: `setup()` is both a RATS hook and an explicitly
  called factory. The automatic invocation opens a connection whose returned
  handle the runner discards. Explicit closes after assertions also don't run
  when an assertion fails.
- `rats/core/119_case_test.rugo`: 1,421 lines cover several distinct topics.
  Shorter files for matching, expression results, scope, and diagnostics would
  be easier to navigate while retaining separate named tests.

## 1. The test name is a claim

Prefer behavior names: "each visits every element in order", "filter leaves
the original array unchanged", "missing callbacks identify the call site".
The name supplies the explanation; the body supplies the evidence.

```ruby
rats "each visits every element in order"
  visited = []

  [10, 20, 30].each(fn(number)
    append visited, number
  end)

  test.assert_eq visited, [10, 20, 30]
end
```

This detects skipped callbacks, missing elements, and reordered visits.
Keep regression references in short comments explaining the historical trap.
Remove obsolete comments and large section banners as files are touched.

## 2. Show the complete expected value

```ruby
rats "map doubles every number and preserves order"
  numbers = [1, 2, 3]

  doubled = numbers.map(fn(n) n * 2 end)

  test.assert_eq doubled, [2, 4, 6]
  test.assert_eq numbers, [1, 2, 3]
end
```

Whole-array assertions catch extra elements as well as missing ones. Use whole
hashes for records and `result.lines` for line-oriented output. For unordered
results, normalize only the ordering the contract leaves unspecified. Never
sort the result of an operation whose ordering is what the test verifies.

When testing JSON content, parse it and assert the structure. When testing JSON
serialization itself, keep exact text assertions where formatting matters.

There is an important exception for a language's own tests:

```ruby
rats "array equality compares nested values"
  left = [1, [2, 3]]
  right = [1, [2, 3]]

  test.assert_eq(left == right, true)
  test.assert_eq(left != right, false)
end
```

Replacing these with `test.assert_eq(left, right)` would stop exercising Rugo's
equality operator: the test module uses its own Go deep-equality check.
Likewise, `assert_true` checks truthiness. Keep `assert_eq(value, true)` when
the contract requires an actual Bool.

## 3. Choose the execution boundary deliberately

| What the test proves | Preferred form |
|---|---|
| Collection results, module values, ordinary runtime errors | Direct Rugo calls |
| Top-level scope, invalid syntax, output, exit status, source locations | `eval.run` with visible source |
| Require resolution, named files, multi-file programs | Files plus `eval.file` or the relevant CLI command |
| CLI parsing, building, emitting Go, invoking external tools | `test.run` |

RATS blocks are function scopes. Moving a top-level closure regression into a
RATS block changes the program being tested. Keep scope-sensitive cases in
source snippets, even when an inline equivalent looks shorter.

Process isolation is also useful for web servers and process-global state.
Don't collapse those tests into one long-lived test process just to save lines.

## 4. Write embedded programs as programs

Use raw squiggly heredocs for embedded Rugo by default. That preserves inner
interpolation and escapes while keeping the source readable.

```ruby
rats "a top-level closure sees the latest assignment"
  source = <<~'RUGO'
    count = 10
    read_count = fn() count end
    count = 20
    puts read_count()
  RUGO

  result = eval.run(source)

  check.success(result)
  test.assert_eq result.lines, ["20"]
end
```

`check.success` here is a prototype helper described below. It checks exit
status and includes the child output on failure.

Use an interpolating heredoc when the outer test intentionally inserts a path
or value. Keep one-line source strings for genuinely one-line programs. A
string full of escaped newlines should usually become a heredoc.

Preserve exact source layout in location tests. Preserve unusual syntax in
syntax regressions. The tested syntax is evidence, not formatting debt.

## 5. A tiny support module with useful failures

Prototype these as ordinary native Rugo functions behind an explicit `require`:

| Helper | Contract |
|---|---|
| `check.success(result)` | Require exit 0; show status and captured output on failure |
| `check.failure(result, text)` | Require nonzero exit and expected text |
| `check.raises(text, action)` | Require an actual runtime error containing the expected text |
| `check.named(label, action)` | Prefix a failed assertion group with the case label |

Keep `eval.run` and `test.run` visible at the call site so readers know what is
being executed. Helpers should remove plumbing while leaving the scenario and
expected value in the test.

```ruby
rats "missing callbacks explain what the caller must supply"
  numbers = [1, 2, 3]

  check.raises(".filter() requires a function argument", fn()
    numbers.filter()
  end)
end
```

The helper must fail if the call succeeds, including when it returns an
error-shaped string. Put only the operation under test in the action, not
assertions that might be mistaken for the expected error. Tests that require
exact error text must retain exact matching.

The native prototypes catch and rethrow messages. They are suitable for the
examples tested here, but don't preserve every runner control signal: a skip
inside an action needs deliberate treatment before these become shared APIs.

Avoid a fluent assertion object or a second language of matchers. Ordinary
functions and the existing assertions are enough for the first pass.

## 6. Named cases instead of anonymous nested loops

Use arrays of named records when the operation is identical and only the data
changes. Arrays make case order explicit; field names explain each column.

```ruby
rats "take handles empty, short, and oversized requests"
  numbers = [10, 20, 30]
  cases = [
    {name: "nothing requested", count: 0, want: []},
    {name: "prefix requested", count: 2, want: [10, 20]},
    {name: "more than available", count: 5, want: [10, 20, 30]}
  ]

  for row in cases
    check.named(row.name, fn()
      test.assert_eq numbers.take(row.count), row.want
    end)
  end
end
```

The prototype still fails fast and counts as one RATS test. It adds context;
it isn't a subtest implementation. Keep separate `rats` blocks for independent
behaviors, different source layouts, or different setup needs.

A future `test.subtest(name, action)` could report and continue through failing
rows. That needs explicit semantics for skip, filtering, recap, counts, and
shared parent fixtures. It is a runner feature, not a clever loop helper.

For a short pair table, destructuring is fine. For three or more fields, prefer
records over positional arrays. Keep expected values literal rather than
computing them with the same operation under test.

## 7. Fixtures are owned, fresh values

Use names such as `new_database()` and `write_module_tree()`. Reserve `setup`,
`teardown`, `setup_file`, and `teardown_file` for actual RATS hooks. A factory
should return a fresh value on each call rather than relying on top-level
variables, which aren't visible inside RATS blocks.

For resources today, a domain-specific callback scope can close on ordinary
success and failure:

```ruby
def with_database(action)
  db = sqlite.open(":memory:")
  result = try action(db) or err
    try sqlite.close(db)
    raise err
  end
  sqlite.close(db)
  return result
end
```

This prototype was checked on success and failure, including attempts to query
the closed connection. The error path preserves the action's failure if cleanup
also fails. Skip before entering this scope: rethrowing an error message doesn't
preserve the runner's special skip signal.

Longer term, `test.cleanup(fn() ... end)` would make ownership clearer without
wrapping every test in a closure. Proposed contract: LIFO cleanup after normal
completion, assertion failure, or skip; preserve the primary failure and report
cleanup failures. Timeout behavior needs separate design because the current
runner cannot stop a timed-out goroutine before cleaning up its resources.

## 8. Concurrency examples should synchronize

```ruby
rats "done becomes true after the task result is collected"
  task = spawn 42

  test.assert_eq task.wait(2), 42
  test.assert_eq task.done, true
end
```

For unfinished work, use a queue as a gate and a timeout as a failure bound.
Release the gate and join the task before asserting saved observations so a
failed assertion doesn't strand the worker. For web tests, binding port 0 and
waiting for `web.port()` is better than guessing a port and sleeping.

Keep timing tests only where timing is the behavior being tested. The book's
fixed-sleep demonstration is an application example, not a test-synchronization
pattern to copy.

## How much book style to adopt

- Use meaningful short names, `result.status`, colon-key records, interpolation
  for generated messages, raw strings for literal content, and `append` sugar.
- Use paren-free assertions when they read clearly. Keep parentheses for nested
  calls and expressions whose boundaries become harder to see without them.
- Use `for` for executing cases. Use collection pipelines when transformation is
  the subject or makes the expected data clearer.
- Use closure factories for real stateful fixtures, rather than constructing
  assertion objects for every test.
- Keep source-level distinctions intact: bare calls, explicit calls, typed and
  untyped code, and top-level versus function scope may be separate regressions.

Some book prose also needs a follow-up: it says append requires explicit
reassignment, describes `try/or` as null coalescing, and shows older output
formats. The book is guidance; executable tests decide current behavior.

## Proposed rollout

1. **Small pilot:** variables, array methods, collection equality, and the two
   newest callback/uniq regressions. Fix assertions that don't establish their
   named behavior. Keep the syntax/CLI/location checks that need isolation.
2. **Support vocabulary:** review the four native helper contracts, including
   deliberately failing helper tests. Introduce one explicit shared module only
   after the pilot shows where it pays for itself.
3. **Stateful suites:** fix SQLite factory naming and resource ownership, then
   audit spawn/queue/web tests for avoidable sleeps and fixture indirection.
4. **Navigation:** split oversized files by behavior. Preserve test names where
   practical so filters and regression history stay useful.
5. **Optional runner work:** evaluate `test.cleanup` first and `test.subtest`
   second. Keep these separate from mechanical readability changes.

Review each migrated test for unchanged or stronger coverage. In particular:
which operation is observed, in which scope, with which types, and at which
execution boundary? Run the affected files, then the complete RATS suite.
Use deliberate broken behaviors on the pilot to check that the new assertions
fail for the reason the test name promises.

## Regression gate for each file

Treat a readability refactor as a change to the test oracle. A green suite
alone cannot tell us whether the new test still detects the old bug.

1. **Establish the baseline.** Record the commit and test names. Build with
   `make build`, then run the original file with `--recap --timing`. Save the
   result and any existing failures before editing.
2. **Map the contract.** For each test, record the operation, input, expected
   value and type, scope, execution boundary, and any source-location or
   ordering requirements. Every old behavior must map to a retained or stronger
   assertion. Keep historical regression inputs and references.
3. **Refactor one file and its owned fixtures.** Keep compiler changes and new
   test-framework APIs in separate work. Audit fixture references before moving
   or removing them. Unusual or intentionally invalid source may need to stay
   exactly as written. Record files reviewed and retained without changes too.
4. **Prove the new test can fail.** For important rewrites, reproduce the old
   bug or introduce a small temporary mutation to the behavior under test in an
   isolated worktree. Examples: skip an each callback, return an extra map
   element, or bypass missing-callback validation. Run the corresponding old
   and new tests where possible. Inspect the failure: a compiler error caused
   by the mutation is not evidence that the intended assertion detected it.
   Merely changing the expected value tests assertion plumbing, not coverage.
5. **Separate coverage repairs from pure refactors.** If the original test
   doesn't detect the mutation, record the existing coverage gap. Strengthen
   the assertion deliberately and show red/green evidence rather than claiming
   unchanged coverage.
6. **Restore and verify.** Remove the mutation, rebuild, rerun the focused file
   and any tests sharing changed helpers, then run `make rats`. Compare names,
   counts, skips, and results with the baseline. Explain every count change;
   fewer named tests or new skips must not silently hide lost coverage.
7. **Record evidence in kata.** Mark a file complete only with its outcome,
   old-to-new coverage notes, mutation or historical-repro result, focused and
   full-suite results, and commit when available. Use one focused commit per
   file or tightly coupled file/fixture unit.

No finite test run proves the absence of regressions. This gate combines
contract review, demonstrated fault detection, and integration checks so a
shorter test has to earn our trust.

## Prototype evidence

Scratch examples are under `/tmp/opencode/rats-style/`. They exercise direct
collection assertions, operator checks, raw snippets, structured JSON, named
cases, callbacks, task synchronization, database cleanup, and helper failures.

- All 20 prototype tests passed together after rebuilding Rugo.
- A nested negative fixture deliberately failed all three tests, and its caller
  checked that the failure messages retained the case label and child output.
- The 36 missing-callback checks took 562 microseconds of runner time together
  in the direct prototype. Their four current eval-based groups took about
  6.6 seconds in the preceding focused run. These are single-run observations,
  not controlled benchmarks; the direct test executable still needs compiling.
- A source-location check remains a separate subprocess test. Direct runtime
  assertions don't replace that coverage.
- Full existing suite: 201 files, 2,628 tests, 2,603 passed, 0 failed, 25 skipped.
