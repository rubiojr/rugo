# Pipe: chain shell → module function → builtin
import "str"
echo "hello world" | str.upper | puts
