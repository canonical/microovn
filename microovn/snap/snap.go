// Package snap implements snap / snapctl specific functions.
package snap

import (
	"fmt"

	"github.com/canonical/lxd/shared"
)

// Start - start snap service as represented by "service" string, optionally
// leaving it enabled for future reboots when "enable" is true.
func Start(service string, enable bool) error {
	args := []string{
		"start",
		fmt.Sprintf("microovn.%s", service),
	}

	if enable {
		args = append(args, "--enable")
	}

	_, err := shared.RunCommand("snapctl", args...)
	if err != nil {
		return err
	}

	return nil
}

// Stop stops specified snap service. Service can be optionally also disabled, ensuring
// that it won't be automatically started on system reboot.
func Stop(service string, disable bool) error {
	args := []string{
		"stop",
		fmt.Sprintf("microovn.%s", service),
	}

	if disable {
		args = append(args, "--disable")
	}

	_, err := shared.RunCommand("snapctl", args...)
	if err != nil {
		return err
	}

	return nil
}

// Restart - restart snap service as represented by "service" string.
func Restart(service string) error {
	args := []string{
		"restart",
		fmt.Sprintf("microovn.%s", service),
	}

	_, err := shared.RunCommand("snapctl", args...)
	if err != nil {
		return err
	}

	return nil
}

// Reload - reload snap service as represented by "service" string.
func Reload(service string) error {
	args := []string{
		"restart",
		"--reload",
		fmt.Sprintf("microovn.%s", service),
	}

	_, err := shared.RunCommand("snapctl", args...)
	if err != nil {
		return err
	}

	return nil
}
