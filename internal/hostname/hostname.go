//go:build linux

package hostname

import (
	"os"
	"golang.org/x/sys/unix"
)

// SetHostname sets the system hostname using unix package
func SetHostname(name string) error {
	return unix.Sethostname([]byte(name))
}

// GetHostname returns the current system hostname
func GetHostname() (string, error) {
	return os.Hostname()
}