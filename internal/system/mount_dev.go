//go:build !linux

package system

import (
	"log"
	config "github.com/TheRealSibasishBehera/init-go/internal/config"
)

// MountEssential is a development stub for non-Linux platforms
func MountEssential(cfg *config.RunConfig) {
	log.Println("[DEV] Skipping mount operations - not running on Linux")
	log.Printf("[DEV] Would mount essential filesystems for hostname: %s", cfg.Hostname)
}