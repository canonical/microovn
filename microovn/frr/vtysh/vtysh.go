// Package vtysh provides abstraction layer for interacting with FRR's vtysh console
package vtysh

import (
	"context"
	"path/filepath"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

// Command is an object representing FRR's vtysh command, or
// chain of commands that can be executed together.
type Command struct {
	commands []string
}

// Add next command to the existing chain. For example,
// if your first command is 'enable', you can follow up by
// adding 'configure' command to enter shell's configuration mode.
func (c *Command) Add(command string) {
	c.commands = append(c.commands, command)
}

// Execute runs complete chain of commands in the FRR's vtysh. Each
// command is executed via separate '-c' option, which result in a
// script-like behavior within single shell.
func (c *Command) Execute(ctx context.Context) (string, error) {
	args := make([]string, len(c.commands)*2)
	for i, command := range c.commands {
		args[i*2] = "-c"
		args[(i*2)+1] = command
	}

	return shared.RunCommandContext(ctx, filepath.Join(paths.Wrappers(), "vtysh"), args...)
}

// NewVtyshCommand initializes Command object. If argument
// firstCommand is non-empty string, it will be inserted as a first
// command of the command chain. If the argument is empty string,
// the command chain will be initialized empty.
func NewVtyshCommand(firstCommand string) Command {
	commands := make([]string, 0)
	if firstCommand != "" {
		commands = append(commands, firstCommand)
	}

	return Command{
		commands: commands,
	}
}
