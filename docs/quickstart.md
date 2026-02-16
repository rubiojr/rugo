# Quickstart

Get up and running with Rugo in minutes.

## Install

```
go install github.com/rubiojr/rugo@latest
```

## Run your first script

```bash
rugo run script.rugo        # compile and run
rugo build script.rugo      # compile to native binary
rugo emit script.rugo       # print generated Go code
rugo doc http             # show module documentation
```

## Guide

1. [Hello World](quickstart/01-hello-world.md)
2. [Variables](quickstart/02-variables.md)
3. [Strings](quickstart/03-strings.md)
4. [Arrays](quickstart/04-arrays.md)
5. [Hashes](quickstart/05-hashes.md)
6. [Control Flow](quickstart/06-control-flow.md)
7. [For Loops](quickstart/07-for-loops.md)
8. [Functions](quickstart/08-functions.md)
9. [Lambdas](quickstart/08b-lambdas.md)
10. [Shell Commands](quickstart/09-shell.md)
11. [Modules](quickstart/10-modules.md)
12. [Error Handling](quickstart/11-error-handling.md)
13. [Concurrency](quickstart/12-concurrency.md)
14. [Testing with RATS](quickstart/13-testing.md)

### Advanced

15. [Custom Modules](quickstart/14-custom-modules.md)
16. [Benchmarks](quickstart/15-benchmarks.md)
17. [Go Bridge](quickstart/16-go-bridge.md)
18. [Structs](quickstart/17-structs.md)
19. [Web Server](quickstart/18-web.md)
20. [Remote Modules](quickstart/19-remote-modules.md)
21. [Doc Comments](quickstart/20-doc-comments.md)
22. [Sandbox](quickstart/21-sandbox.md)
23. [Go Modules via Require](quickstart/22-go-modules.md)

## Standard Library

The Rugo stdlib (`use` modules) is the idiomatic way to work in Rugo. It provides a curated, Ruby-inspired API covering common tasks — math, file paths, encoding, crypto, time, and more.

```ruby
use "math"
use "filepath"
use "base64"

puts math.sqrt(144.0)                          # 12
puts filepath.join("home", "user", "docs")     # home/user/docs
puts base64.encode("Hello, Rugo!")             # SGVsbG8sIFJ1Z28h
```

Prefer `use` modules for standard operations. They are designed for Rugo and follow its conventions. The `import` keyword (Go bridge) is available for advanced use cases when you need direct access to Go's standard library, but `use` modules are the recommended, idiomatic approach.

See the full list of available modules in the [Modules](modules.md) reference.
