// Package snap implements snap / snapctl specific functions.
package snap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/canonical/lxd/shared"
)

// snapctlRetryTimeout bounds how long a snapctl command is retried when it
// fails because another snap change (install/refresh/connect) is in progress.
// During a snap refresh the daemon's OnStart hook runs snapctl to (re)start
// services before snapd's own service-control/refresh-snap change has settled,
// so the nested snapctl can transiently fail for a while (the pre-refresh hook
// alone can take ~60s). Two minutes gives a comfortable margin.
const snapctlRetryTimeout = 2 * time.Minute

// snapctlRetryCooldown is the pause between retry attempts.
const snapctlRetryCooldown = 2 * time.Second

// isChangeInProgress reports whether err is a transient "another snap change is
// in progress" failure, rather than a real error that should be surfaced
// immediately. snapd only reports this on snapctl's stderr, which the
// shared.RunError captures, so we match on the message text.
func isChangeInProgress(err error) bool {
	if err == nil {
		return false
	}

	var runErr shared.RunError
	if errors.As(err, &runErr) {
		return strings.Contains(runErr.StdErr().String(), "change in progress")
	}

	return strings.Contains(err.Error(), "change in progress")
}

// runSnapctl runs a snapctl command, retrying while another snap change is in
// progress (see isChangeInProgress), up to snapctlRetryTimeout. Any other
// failure is returned immediately.
func runSnapctl(ctx context.Context, args ...string) error {
	retryCtx, cancel := context.WithTimeout(ctx, snapctlRetryTimeout)
	defer cancel()

	opts := &shared.RunCommandRetryOpts{
		RetryFunc: func(_ uint, err error) bool {
			if !isChangeInProgress(err) {
				return false
			}

			time.Sleep(snapctlRetryCooldown)
			return true
		},
	}

	_, err := shared.RunCommandRetry(retryCtx, opts, "snapctl", args...)
	return err
}

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

	return runSnapctl(context.Background(), args...)
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

	_, err := shared.RunCommand(context.Background(), "snapctl", args...)
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

	_, err := shared.RunCommand(context.Background(), "snapctl", args...)
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

	_, err := shared.RunCommand(context.Background(), "snapctl", args...)
	if err != nil {
		return err
	}

	return nil
}
