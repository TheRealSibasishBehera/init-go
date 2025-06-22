package exec

import (
	"sync"
	"testing"
)

func TestExecuteCommand_Success(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"echo", "hello world"},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	response, err := ExecuteCommand(req, envs, mutex)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %v", response.ExitCode)
	}

	if response.ExitSignal != nil {
		t.Errorf("Expected no exit signal, got: %v", response.ExitSignal)
	}

	expectedOutput := "hello world\n"
	if string(response.Stdout) != expectedOutput {
		t.Errorf("Expected stdout '%s', got '%s'", expectedOutput, string(response.Stdout))
	}

	if len(response.Stderr) != 0 {
		t.Errorf("Expected empty stderr, got: %s", string(response.Stderr))
	}
}

func TestExecuteCommand_Failure(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"ls", "/nonexistent-directory"},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	response, err := ExecuteCommand(req, envs, mutex)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code, got: %v", response.ExitCode)
	}

	if response.ExitSignal != nil {
		t.Errorf("Expected no exit signal, got: %v", response.ExitSignal)
	}

	if len(response.Stdout) != 0 {
		t.Errorf("Expected empty stdout, got: %s", string(response.Stdout))
	}

	if len(response.Stderr) == 0 {
		t.Error("Expected stderr output for failed command")
	}
}

func TestExecuteCommand_WithEnvironment(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"sh", "-c", "echo $TEST_VAR"},
	}
	envs := map[string]string{
		"TEST_VAR": "test_value",
	}
	mutex := &sync.Mutex{}

	response, err := ExecuteCommand(req, envs, mutex)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %v", response.ExitCode)
	}

	expectedOutput := "test_value\n"
	if string(response.Stdout) != expectedOutput {
		t.Errorf("Expected stdout '%s', got '%s'", expectedOutput, string(response.Stdout))
	}
}

func TestExecuteCommand_StderrOutput(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"sh", "-c", "echo 'error message' >&2"},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	response, err := ExecuteCommand(req, envs, mutex)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %v", response.ExitCode)
	}

	if len(response.Stdout) != 0 {
		t.Errorf("Expected empty stdout, got: %s", string(response.Stdout))
	}

	expectedStderr := "error message\n"
	if string(response.Stderr) != expectedStderr {
		t.Errorf("Expected stderr '%s', got '%s'", expectedStderr, string(response.Stderr))
	}
}

func TestExecuteCommand_EmptyCommand(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	// should panic or return an error as no command is provided
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty command, but didn't panic")
		}
	}()

	ExecuteCommand(req, envs, mutex)
}

func TestExecuteCommand_MutexSerialization(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"echo", "test"},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	// run multiple commands concurrently to test mutex
	results := make(chan ExecResponse, 3)
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			response, err := ExecuteCommand(req, envs, mutex)
			results <- response
			errors <- err
		}()
	}

	for i := 0; i < 3; i++ {
		response := <-results
		err := <-errors

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if response.ExitCode == nil || *response.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got: %v", response.ExitCode)
		}

		expectedOutput := "test\n"
		if string(response.Stdout) != expectedOutput {
			t.Errorf("Expected stdout '%s', got '%s'", expectedOutput, string(response.Stdout))
		}
	}
}

func TestEnvToSlice(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		expected []string
	}{
		{
			name:     "empty environment",
			envs:     map[string]string{},
			expected: []string{},
		},
		{
			name: "single environment variable",
			envs: map[string]string{
				"KEY": "value",
			},
			expected: []string{"KEY=value"},
		},
		{
			name: "multiple environment variables",
			envs: map[string]string{
				"PATH": "/bin:/usr/bin",
				"HOME": "/home/user",
			},
			expected: []string{"PATH=/bin:/usr/bin", "HOME=/home/user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := envToSlice(tt.envs)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d environment variables, got %d", len(tt.expected), len(result))
				return
			}

			resultMap := make(map[string]bool)
			for _, env := range result {
				resultMap[env] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected environment variable '%s' not found in result", expected)
				}
			}
		})
	}
}

func TestExecuteCommand_WithMultipleArgs(t *testing.T) {
	req := ExecRequest{
		Cmd: []string{"sh", "-c", "echo $1 $2", "_", "hello", "world"},
	}
	envs := map[string]string{}
	mutex := &sync.Mutex{}

	response, err := ExecuteCommand(req, envs, mutex)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %v", response.ExitCode)
	}

	expectedOutput := "hello world\n"
	if string(response.Stdout) != expectedOutput {
		t.Errorf("Expected stdout '%s', got '%s'", expectedOutput, string(response.Stdout))
	}
}

