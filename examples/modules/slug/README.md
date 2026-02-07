# Rugo Slug Module (External Module Example)

This example demonstrates how to create a custom Rugo module that wraps an
external Go library — in this case, [gosimple/slug](https://github.com/gosimple/slug)
for generating URL-friendly slugs from text.

## Structure

```
slug/
  module/          # The Rugo module package (reusable by anyone)
    slug.go        # Module registration
    runtime.go     # Go runtime (struct + methods)
  custom-rugo/     # A custom Rugo binary that includes the slug module
    main.go        # Thin wrapper: stdlib + slug module
  example.rg       # Example Rugo script using the slug module
```

## Build & Run

```bash
cd custom-rugo
go build -o myrugo .
./myrugo ../example.rg
```

## How It Works

1. `module/` is a standard Rugo module — it calls `modules.Register()` in `init()`
   just like the built-in modules.
2. `custom-rugo/main.go` imports the slug module alongside the standard Rugo
   modules, then calls `cmd.Execute()`.
3. The resulting binary is a full Rugo compiler with the `slug` module available
   for `import "slug"` in `.rg` scripts.
