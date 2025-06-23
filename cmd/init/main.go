package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/TheRealSibasishBehera/init-go/internal/config"
	"github.com/TheRealSibasishBehera/init-go/internal/server"
	"github.com/TheRealSibasishBehera/init-go/internal/system"
)

const DefaultConfigPath = "/fly/run.json"

var waitPidMutex sync.Mutex

func main() {
	if err := validatePID1(); err != nil {
		log.Fatalf("FATAL: Not running as PID 1: %v", err)
	}

	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("FATAL: Configuration error: %v", err)
	}
	log.Printf("Loaded configuration: hostname=%s", cfg.Hostname)

	system.MountEssential(cfg)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGCHLD, syscall.SIGINT)

	go func() {
		server.StartVSocServer(&waitPidMutex, cfg.ExtraEnv)
	}()
	log.Printf("Started VSOCK server on port %d", server.VSockPort)

	var cmd *exec.Cmd
	if command := cfg.GetCommand(); len(command) > 0 {
		cmd = exec.Command(command[0], command[1:]...)
		if env := cfg.GetEnvironment(); len(env) > 0 {
			cmd.Env = append(os.Environ(), env...)
		}
		cmd.Dir = cfg.GetWorkingDir()

		if err := cmd.Start(); err != nil {
			log.Fatalf("FATAL: Failed to start command %v: %v", command, err)
		}
		log.Printf("Started application %v with PID %d", command, cmd.Process.Pid)
	} else {
		log.Fatalf("FATAL: No command specified in configuration")
	}

	log.Println("Init system ready, entering main loop...")

	for sig := range signals {
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("Received %s, shutting down gracefully...", sig)
			if cmd != nil && cmd.Process != nil {
				log.Printf("Terminating child process %d", cmd.Process.Pid)
				cmd.Process.Signal(syscall.SIGTERM)
			}
			os.Exit(0)

		case syscall.SIGCHLD:
			log.Println("Received SIGCHLD, reaping zombies...")
			reapZombies(&waitPidMutex)
		}
	}
}

func reapZombies(waitPidMutex *sync.Mutex) {
	waitPidMutex.Lock()
	defer waitPidMutex.Unlock()

	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if err != nil || pid == 0 {
			break
		}
		log.Printf("Reaped zombie process %d with status %d", pid, status)
	}
}

func loadConfiguration() (*config.RunConfig, error) {
	cfg, err := config.LoadConfig(DefaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from %s: %w", DefaultConfigPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func validatePID1() error {
	pid := os.Getpid()
	if pid != 1 {
		return fmt.Errorf("must run as PID 1, currently running as PID %d", pid)
	}
	log.Println("Validated: Running as PID 1 - container init mode")
	return nil
}
