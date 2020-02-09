// +build linux
// +build amd64 arm64

package ioengine

import "unsafe"

type Iocb struct {
	Data   unsafe.Pointer
	Key    uint64
	Opcode int16
	Prio   int16
	Fd     uint32
	Buf    unsafe.Pointer
	Nbytes uint64
	Offset int64
	pad1   int64
	Flags  uint32
	Resfd  uint32
}

type Event struct {
	Data unsafe.Pointer
	Obj  *Iocb
	Res  int64
	Res2 int64
}
