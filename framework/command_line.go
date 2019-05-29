package framework

import (
	"fmt"
)

// CommandLineRunner collects methods for running commands
type CommandLineRunner struct {
}

// NewCommandLineRunner returns a new CommandLineRunner
func NewCommandLineRunner() *CommandLineRunner {
	return nil
}

// Run executes a command
func (r *CommandLineRunner) Run(details *commandDetails, sourceCode string) error {
	return fmt.Errorf("IMPLEMENT ME")
}
