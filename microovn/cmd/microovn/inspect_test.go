package main

import (
	"errors"
	"testing"
)

func TestInspectRejectsInvalidInvocation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "positional argument", args: []string{"node1"}},
		{name: "verbose JSON", args: []string{"--verbose", "--format=json"}},
		{name: "invalid format", args: []string{"--format=yaml"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common := &CmdControl{}
			command := (&cmdInspect{common: common}).Command()
			command.SetArgs(test.args)
			command.SilenceErrors = true
			command.SilenceUsage = true

			err := command.Execute()
			if err == nil {
				t.Fatal("inspect accepted an invalid invocation")
			}

			if common.ExitCode != 2 {
				t.Fatalf("inspect exit code is %d, want 2", common.ExitCode)
			}
			if got := commandExitCode(err, common.ExitCode); got != 2 {
				t.Fatalf("commandExitCode returned %d, want 2", got)
			}
		})
	}
}

func TestCommandExitCode(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		reportExitCode int
		want           int
	}{
		{name: "healthy report", reportExitCode: 0, want: 0},
		{name: "report finding", reportExitCode: 1, want: 1},
		{name: "other command error", err: errors.New("failed"), want: 1},
		{name: "configured command error", err: errors.New("failed"), reportExitCode: 2, want: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := commandExitCode(test.err, test.reportExitCode); got != test.want {
				t.Fatalf("commandExitCode returned %d, want %d", got, test.want)
			}
		})
	}
}
