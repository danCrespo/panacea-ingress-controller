package cmdline

import (
	"os"
	"testing"
)

func TestCommandLineArguments(t *testing.T) {
	t.Name()
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"controller", "start"}, ""},
		{[]string{"controller", "--version"}, "Version 1.0"},
		{[]string{"controller", "invalid"}, "Error: invalid argument"},
	}

	for _, test := range tests {
		os.Args = test.args
		result := runCommandLine()
		err := result.Execute()
		if err != nil {
			t.Errorf("For args %v, Execute() returned an error: %v", test.args, err)
		}
		if err.Error() != test.expected {
			t.Errorf("For args %v, expected %v but got %v", test.args, test.expected, err.Error())
		}
	}
}

func runCommandLine() CmdLine {
	return New()
}

func TestStartCommand(t *testing.T) {
	t.Name()
	cmdLine := New()

	err := cmdLine.Execute()
	if err != nil {
		t.Errorf("Execute() returned an error: %v", err)
	}

}
