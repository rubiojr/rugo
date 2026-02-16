# time

Time operations: timestamps, sleeping, formatting, and parsing.

```ruby
use "time"
```

## now

Return the current Unix timestamp as a float with nanosecond precision.

```ruby
t = time.now()   # e.g. 1720000000.123456789
```

## sleep

Sleep for the given number of seconds (float).

```ruby
time.sleep(0.5)   # sleep 500ms
time.sleep(2.0)   # sleep 2 seconds
```

## format

Format a Unix timestamp using a Go time layout string.

```ruby
result = time.format(0.0, "2006-01-02")   # "1970-01-01"
```

## parse

Parse a time string using a Go time layout, returning a Unix timestamp float. Panics on invalid input.

```ruby
t = time.parse("2024-01-15", "2006-01-02")
```

## since

Return seconds elapsed since the given Unix timestamp.

```ruby
start = time.now()
time.sleep(0.1)
elapsed = time.since(start)   # ~0.1
```

## millis

Return the current time in milliseconds as an integer.

```ruby
ms = time.millis()   # e.g. 1720000000123
```
