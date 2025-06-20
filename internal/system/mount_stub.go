//go:build !linux

package system

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func MountEssential() {
	log.Println("[DEV] Skipping mount operations - development mode")
	
	setHostname("lab-vm")
	
	fmt.Printf("Lab starting /bin/sh\n")
	
	cmd := exec.Command("/bin/sh")
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
	log.Printf("[DEV] Would set hostname to: %s", hostname)
}

func mount(source, target, filesystemtype string, flags uintptr) {
	log.Printf("[DEV] Would mount: %s -> %s (%s)", source, target, filesystemtype)
}