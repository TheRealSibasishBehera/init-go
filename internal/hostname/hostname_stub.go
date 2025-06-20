//go:build !linux

package hostname

import (
	"log"
	"os"
)

// SetHostname sets the system hostname (development stub)
func SetHostname(name string) error {
	log.Printf("[DEV] Would set hostname to: %s", name)
	return nil
}

// GetHostname returns the current system hostname
func GetHostname() (string, error) {
	return os.Hostname()
}