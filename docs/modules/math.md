# math

Mathematical functions and constants.

```ruby
use "math"
```

## abs

Return the absolute value of n.

```ruby
math.abs(-5.0)   # 5.0
math.abs(3.0)    # 3.0
```

## ceil

Round n up to the nearest integer.

```ruby
math.ceil(2.3)   # 3
math.ceil(-1.5)  # -1
```

## floor

Round n down to the nearest integer.

```ruby
math.floor(2.7)   # 2
math.floor(-1.5)  # -2
```

## round

Round n to the nearest integer.

```ruby
math.round(2.5)   # 3
math.round(2.4)   # 2
```

## max

Return the larger of a and b.

```ruby
math.max(3.0, 5.0)   # 5.0
```

## min

Return the smaller of a and b.

```ruby
math.min(3.0, 5.0)   # 3.0
```

## pow

Return base raised to the power of exp.

```ruby
math.pow(2.0, 10.0)   # 1024.0
```

## sqrt

Return the square root of n.

```ruby
math.sqrt(144.0)   # 12.0
```

## log

Return the natural logarithm of n.

```ruby
math.log(1.0)   # 0.0
```

## log2

Return the base-2 logarithm of n.

```ruby
math.log2(8.0)   # 3.0
```

## log10

Return the base-10 logarithm of n.

```ruby
math.log10(100.0)   # 2.0
```

## sin

Return the sine of n (radians).

```ruby
math.sin(0.0)   # 0.0
```

## cos

Return the cosine of n (radians).

```ruby
math.cos(0.0)   # 1.0
```

## tan

Return the tangent of n (radians).

```ruby
math.tan(0.0)   # 0.0
```

## pi

Return the value of Pi.

```ruby
math.pi()   # 3.141592653589793
```

## e

Return the value of Euler's number.

```ruby
math.e()   # 2.718281828459045
```

## inf

Return positive infinity.

```ruby
math.inf()   # +Inf
```

## nan

Return NaN (not a number).

```ruby
math.nan()   # NaN
```

## is_nan

Return true if n is NaN.

```ruby
math.is_nan(math.nan())   # true
math.is_nan(1.0)          # false
```

## is_inf

Return true if n is infinite.

```ruby
math.is_inf(math.inf())   # true
math.is_inf(1.0)          # false
```

## clamp

Clamp n between min and max.

```ruby
math.clamp(5.0, 0.0, 10.0)    # 5.0
math.clamp(-5.0, 0.0, 10.0)   # 0.0
math.clamp(15.0, 0.0, 10.0)   # 10.0
```

## random

Return a random float in [0.0, 1.0).

```ruby
r = math.random()   # e.g. 0.6046602879796196
```

## random_int

Return a random integer in [min, max).

```ruby
r = math.random_int(1, 10)   # e.g. 7
```
