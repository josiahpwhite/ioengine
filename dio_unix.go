// +build linux

package ioengine

import (
	"os"
	"syscall"
)

const (
	// AlignSize size to align the buffer
	AlignSize = 512
)

// OpenFileWithDIO open files with O_DIRECT flag
func OpenFileWithDIO(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, syscall.O_DIRECT|flag, perm)
}