// +build linux darwin freebsd openbsd netbsd dragonfly

package flock

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// Acquire acquire a file lock, will not block
func Acquire(file string) error {

	fd, err := unix.Open(file, unix.O_CREAT|unix.O_RDWR, 0600)
	if err != nil {
		return err
	}

	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return err
	}

	// Write pid to file
	_, _ = unix.Write(fd, []byte(strconv.Itoa(unix.Getpid())))
	return nil
}
