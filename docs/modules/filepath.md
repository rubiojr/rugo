# filepath

File path manipulation and querying.

```ruby
use "filepath"
```

## join

Join path segments into a single path.

```ruby
filepath.join("a", "b", "c")             # "a/b/c"
filepath.join("/usr", "local", "bin")     # "/usr/local/bin"
```

## base

Return the last element of a path.

```ruby
filepath.base("/foo/bar/baz.txt")   # "baz.txt"
filepath.base("/foo/bar/")          # "bar"
```

## dir

Return all but the last element of a path.

```ruby
filepath.dir("/foo/bar/baz.txt")   # "/foo/bar"
filepath.dir("foo/bar")            # "foo"
```

## ext

Return the file extension.

```ruby
filepath.ext("file.txt")         # ".txt"
filepath.ext("archive.tar.gz")   # ".gz"
filepath.ext("file")             # ""
```

## abs

Return the absolute path. Panics on error.

```ruby
p = filepath.abs(".")
filepath.is_abs(p)   # true
```

## rel

Return a relative path from base to target. Panics on error.

```ruby
filepath.rel("/a/b", "/a/b/c/d")   # "c/d"
```

## glob

Return files matching a glob pattern.

```ruby
matches = filepath.glob("/tmp/*")
```

## clean

Return the shortest equivalent path.

```ruby
filepath.clean("a/b/../c")    # "a/c"
filepath.clean("a//b///c")    # "a/b/c"
```

## is_abs

Return true if the path is absolute.

```ruby
filepath.is_abs("/foo")   # true
filepath.is_abs("foo")    # false
```

## split

Split a path into directory and file components. Returns a two-element array.

```ruby
parts = filepath.split("/foo/bar.txt")
parts[0]   # "/foo/"
parts[1]   # "bar.txt"
```

## match

Return true if the name matches the glob pattern. Panics on error.

```ruby
filepath.match("*.txt", "file.txt")   # true
filepath.match("*.go", "file.txt")    # false
```
