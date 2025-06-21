//go:build linux

package system

import (
	"golang.org/x/sys/unix"
	"os"
)

// SetHostname sets the system hostname using unix package
func SetHostname(name string) error {
	return unix.Sethostname([]byte(name))
}

// GetHostname returns the current system hostname
func GetHostname() (string, error) {
	return os.Hostname()
}

