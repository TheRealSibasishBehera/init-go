//go:build linux

package system

import (
	"fmt"
	config "github.com/TheRealSibasishBehera/init-go/internal/config"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
)

func MountEssential(config *config.RunConfig) {
	if err := mount("none", "/proc", "proc", CommonMountFlags, ""); err != nil {
		panic(fmt.Sprintf("failed to mount /proc: %v", err))
	}
	if err := mount("none", "/dev/pts", "devpts", MS_NOSUID|MS_NOEXEC, "mode=0620,gid=5,ptmxmode=666"); err != nil {
		panic(fmt.Sprintf("failed to mount /dev/pts: %v", err))
	}
	if err := mount("none", "/dev/mqueue", "mqueue", CommonMountFlags, ""); err != nil {
		panic(fmt.Sprintf("failed to mount /dev/mqueue: %v", err))
	}
	if err := mount("none", "/dev/shm", "tmpfs", MS_NOSUID|MS_NODEV, ""); err != nil {
		panic(fmt.Sprintf("failed to mount /dev/shm: %v", err))
	}
	if err := mount("none", "/sys", "sysfs", CommonMountFlags, ""); err != nil {
		panic(fmt.Sprintf("failed to mount /sys: %v", err))
	}
	if err := mount("none", "/sys/fs/cgroup", "cgroup", CommonMountFlags, ""); err != nil {
		panic(fmt.Sprintf("failed to mount /sys/fs/cgroup: %v", err))
	}

	setHostname(config.Hostname)

	cmd := exec.Command("/bin/sh")

	// cmd.Env = append(cmd.Env, paths)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("could not start /bin/sh, error: %s", err))
	}

	err = cmd.Wait()
	if err != nil {
		panic(fmt.Sprintf("could not wait for /bin/sh, error: %s", err))
	}
}
func setHostname(hostname string) {
	err := unix.Sethostname([]byte(hostname))
	if err != nil {
		panic(fmt.Sprintf("cannot set hostname to %s, error: %s", hostname, err))
	}
}

const (
	MS_NODEV    = unix.MS_NODEV
	MS_NOEXEC   = unix.MS_NOEXEC
	MS_NOSUID   = unix.MS_NOSUID
	MS_RELATIME = unix.MS_RELATIME

	CommonMountFlags = MS_NODEV | MS_NOEXEC | MS_NOSUID
)

func mount(source, target, filesystemtype string, flags uintptr, data string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("failed to create mount target %s: %w", target, err)
		}
	}

	if err := unix.Mount(source, target, filesystemtype, flags, data); err != nil {
		return fmt.Errorf("failed to mount %s to %s (%s): %w", source, target, filesystemtype, err)
	}
	return nil
}
