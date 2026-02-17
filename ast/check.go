package ast

// Check validates an AST without modifying it.
type Check interface {
	Name() string
	Check(prog *Program) error
}

// CheckChain runs checks in order, stopping at the first error.
type CheckChain []Check

// Run executes each check in sequence. Returns nil if all pass.
func (cc CheckChain) Run(prog *Program) error {
	for _, c := range cc {
		if err := c.Check(prog); err != nil {
			return err
		}
	}
	return nil
}
