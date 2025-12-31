package core

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// EvaluateCondition compiles and evaluates a string expression against the SystemContext.
// Returns true if the condition is met (or empty), false otherwise.
func EvaluateCondition(condition string, ctx *SystemContext) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// Compile the expression
	// We pass the ctx struct directly so fields like OS, Hardware.GPUVendor can be accessed.
	program, err := expr.Compile(condition, expr.Env(ctx))
	if err != nil {
		return false, fmt.Errorf("invalid condition '%s': %v", condition, err)
	}

	// Run the expression
	output, err := expr.Run(program, ctx)
	if err != nil {
		return false, fmt.Errorf("evaluation failed: %v", err)
	}

	// Check result type
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("condition must return a boolean, got %T", output)
	}

	return result, nil
}
