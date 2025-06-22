package exec

import (
	"bytes"
	"os/exec"
	"sync"
	"syscall"
)

type ExecRequest struct {
	Cmd []string `json:"cmd" validate:"required,min=1"`
}

type ExecResponse struct {
	ExitCode   *int   `json:"exit_code"`
	ExitSignal *int   `json:"exit_signal"`
	Stdout     []byte `json:"stdout"`
	Stderr     []byte `json:"stderr"`
}

func ExecuteCommand(req ExecRequest, envs map[string]string, waitPidMutex *sync.Mutex) (ExecResponse, error) {
	cmd := exec.Command(req.Cmd[0], req.Cmd[1:]...)
	cmd.Env = envToSlice(envs)
	waitPidMutex.Lock()
	defer waitPidMutex.Unlock()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// no handling error or early return
	// we want to capture the exit code ,signal , output
	_ = cmd.Run()

	var exitCode, exitSignal *int
	if cmd.ProcessState != nil {
		status := cmd.ProcessState.Sys().(syscall.WaitStatus)
		if status.Signaled() {
			signal := int(status.Signal())
			exitSignal = &signal
		} else if status.Exited() {
			code := status.ExitStatus()
			exitCode = &code
		} else {
			exitCode = nil
			exitSignal = nil
		}
	}

	return ExecResponse{
		ExitCode:   exitCode,
		ExitSignal: exitSignal,
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
	}, nil

}

func envToSlice(envs map[string]string) []string {
	envSlice := make([]string, 0, len(envs))
	for key, value := range envs {
		envSlice = append(envSlice, key+"="+value)
	}
	return envSlice
}
