package ovn

import (
	"fmt"

	"github.com/canonical/lxd/shared"
)

func snapStart(service string, enable bool) error {
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

// snapStop stops specified snap service. Service can be optionally also disabled, ensuring
// that it won't be automatically started on system reboot.
func snapStop(service string, disable bool) error {
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

func snapRestart(service string) error {
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

func snapReload(service string) error {
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
