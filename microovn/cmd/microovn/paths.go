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
		Args:  cobra.RangeArgs(0, 1),
		RunE:  c.Run,
	}
	return cmd
}

func (c *cmdPath) Run(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		for k, v := range paths.PathsMap {
			fmt.Printf("%s: %s\n", k, v)
		}
		return nil
	}

	name := args[0]
	path, exists := paths.PathsMap[name]
	if !exists {
		return fmt.Errorf("unknown path name: %s", name)
	}
	fmt.Println(path)
	return nil
}
