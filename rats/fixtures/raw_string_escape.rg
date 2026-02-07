# Raw string escape sequences stay literal
puts '\x1b[32mgreen\x1b[0m'
puts '\033[31mred\033[0m'
puts '\n\t\r'
