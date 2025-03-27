package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/ovn/paths"
)

type cmdPath struct {
	common *CmdControl
}

func (c *cmdPath) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path <NAME>",
		Short: "Retrieve the path for the given name.",
		Args:  cobra.ExactArgs(1),
		RunE:  c.Run,
	}
	return cmd
}

func (c *cmdPath) Run(_ *cobra.Command, args []string) error {
	name := args[0]
	path := paths.GetPath(name)
	if path == nil {
		return fmt.Errorf("unknown path name: %s", name)
	}
	fmt.Println(path)
	return nil
}
