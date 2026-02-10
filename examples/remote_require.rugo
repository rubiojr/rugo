# Remote require: load Rugo modules from git repositories
#
# Syntax: require "host/owner/repo@version" as "alias"
#
# The compiler clones the repo, finds the entry .rg file, and
# merges it into your program — just like a local require.
#
# Versions can be:
#   @v1.0.0   — git tag (cached forever, immutable)
#   @main     — branch (re-fetched each build)
#   @abc1234  — commit SHA (cached forever)
#   (none)    — default branch

require "github.com/rubiojr/rugo-hello@v0.1.0" as "hello"

hello.say("world")
