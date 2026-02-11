---
name: idiomatic-rugo
description: Guide to writing clean, idiomatic Rugo code. Load when writing .rugo scripts, reviewing Rugo code style, or learning Rugo patterns and best practices.
---

# Idiomatic Rugo

A guide to writing clean, expressive Rugo code. Rugo borrows Ruby's elegance, Go's compilation speed, and Bash's street smarts — compiled to native binaries.

This skill covers idioms and patterns that experienced Rugo programmers reach for instinctively. Each chapter is available as a detailed reference.

## Chapters

1. [Expressive Basics](references/01-expressive-basics.md) — String interpolation, paren-free calls, constants, heredocs, and raw strings.
2. [Collections with Character](references/02-collections-with-character.md) — Hashes as lightweight objects, dot access, the factory pattern, array idioms, and iteration.
3. [Functions and Lambdas](references/03-functions-and-lambdas.md) — First-class functions, closures, higher-order patterns, dispatch tables, and composition.
4. [Graceful Failure](references/04-graceful-failure.md) — The three levels of `try/or`, raising errors, and building resilient code.
5. [Shell Power](references/05-shell-power.md) — Shell fallback, backtick capture, the pipe operator, and mixing paradigms.
6. [Concurrent Thinking](references/06-concurrent-thinking.md) — Spawn, parallel, timeouts, and task polling.
7. [Structuring Your Code](references/07-structuring-your-code.md) — Structs, modules, the Go bridge, and type introspection.
8. [Patterns in the Wild](references/08-patterns-in-the-wild.md) — Data pipelines, builder pattern, inline tests, and lessons from real libraries.

## The Rugo Philosophy

1. **Start simple.** Hashes before structs. Functions before classes. Graduate to more structure only when needed.
2. **Be explicit about failure.** Use `try/or` at the right level. Don't let errors propagate silently, and don't panic unnecessarily.
3. **Compose small pieces.** Collection methods, lambdas, and closures combine into powerful patterns without complex abstractions.
4. **Use the shell.** Don't rewrite `curl` or `grep` in Rugo. Shell out for what the shell does best, then process results in Rugo.
5. **Test where you code.** Inline `rats` blocks keep tests and implementation together. Use them.

## Quick Reference

- **Interpolation:** `"Hello, #{name}!"` — any expression inside `#{}`
- **Constants:** Uppercase start → assigned once: `PI = 3.14`
- **Hash objects:** `{name: "Alice", age: 30}` with dot access `person.name`
- **Factory pattern:** Functions returning hashes with lambda methods
- **Error handling:** `try expr` / `try expr or default` / `try expr or err ... end`
- **Shell capture:** `` user = `whoami` ``
- **Pipes:** `echo "hello" | str.upper | puts`
- **Concurrency:** `spawn expr` + `.value` / `parallel ... end`
- **Imports:** `use "http"` (stdlib) / `import "strings"` (Go) / `require "file"` (user)
- **Collection pipelines:** `.filter(fn(x) ... end).map(fn(x) ... end).join(", ")`
- **Inline tests:** `rats "name" ... end` blocks, run with `rugo rats`
- **Internal fields:** `__double_underscore__` convention for private state
