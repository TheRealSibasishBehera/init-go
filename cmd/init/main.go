package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/TheRealSibasishBehera/init_go/internal/config"
)

const DefaultConfigPath = "/fly/run.json"

func main() {
	if err := validatePID1(); err != nil {
		log.Fatalf("Error validating PID 1: %v", err)
	}

	cfg := loadConfiguration()
	log.Printf("Loaded configuration: hostname=%s, tty=%t", cfg.Hostname, cfg.TTY)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGCHLD, syscall.SIGINT)

	var cmd *exec.Cmd
	if command := cfg.GetCommand(); len(command) > 0 {
		cmd = exec.Command(command[0], command[1:]...)

		if env := cfg.GetEnvironment(); len(env) > 0 {
			cmd.Env = append(os.Environ(), env...)
		}

		cmd.Dir = cfg.GetWorkingDir()

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start command %v: %v", command, err)
		} else {
			log.Printf("Started application %v with PID %d", command, cmd.Process.Pid)
		}
	} else {
		log.Println("No command configured to run")
	}

	log.Println("Init process started, waiting for signals...")

	// Main signal handling loop
	for sig := range signals {
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("Received %s, shutting down gracefully...", sig)

			// Terminate child processes
			if cmd != nil && cmd.Process != nil {
				log.Printf("Terminating child process %d", cmd.Process.Pid)
				cmd.Process.Signal(syscall.SIGTERM)
			}

			// Exit gracefully
			os.Exit(0)

		case syscall.SIGCHLD:
			log.Println("Received SIGCHLD, child process state changed.")
			// Reap all available zombie children
			for {
				var status syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
				if err != nil || pid == 0 {
					break
				}
				log.Printf("Reaped child process %d with status %d", pid, status)
			}
		}
	}
}

func loadConfiguration() *config.RunConfig {
	// Try to load from standard path first
	cfg, err := config.LoadConfig(DefaultConfigPath)
	if err != nil {
		// Try local path for development/testing
		cfg, err = config.LoadConfig("./fly_run.json")
		if err != nil {
			log.Printf("No configuration file found, using defaults: %v", err)
			// Return default configuration with fallback to /usr/bin/myapp
			return &config.RunConfig{
				ImageConfig: &config.ImageConfig{
					Cmd: []string{"/usr/bin/myapp"},
				},
				TTY:      false,
				Hostname: "localhost",
			}
		}
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		// Use default configuration
		return &config.RunConfig{
			ImageConfig: &config.ImageConfig{
				Cmd: []string{"/usr/bin/myapp"},
			},
			TTY:      false,
			Hostname: "localhost",
		}
	}

	return cfg
}

func validatePID1() error {
	pid := os.Getpid()
	log.Printf("Init process starting with PID: %d", pid)
	// Accept any PID for demonstration - in production you'd be more strict
	if pid == 1 {
		log.Printf("Running as PID 1 - direct container init")
	} else {
		log.Printf("Running as PID %d - managed by init system", pid)
	}
	return nil
}

