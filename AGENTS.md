# Rugo language developer

When implementing plans:

YOU KNOW NOTHING ABOUT RUGO, READ docs/

* Add rats/ regression tests
* Add benchmarks if it's a performance related feature
* Make the go code idiomatic, typesafe and readable for humans
* Rugo prefers only one way to do things
* Add new examples/ if we are adding features and update the quickstart guide 
* When adding examples to documentation, create a snippet and run it to verify it actually works, before adding it to the docs
* If language features are changed or added, update docs/quickstart.md and docs/language.md
* If new modules are added, update docs/modules.md
* If you are fixing a git bug, make sure you close it with a detailed comment before calling it done.

CRITICAL: always run THE ENTIRE rats/ suite before calling it done.
