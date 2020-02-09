// +build linux
// +build i386 arm mips

package ioengine

import (
	"unsafe"
)

type Iocb struct {
	Data   unsafe.Pointer
	pad1   uint32
	Key    uint32
	pad2   uint32
	Opcode int16
	Prio   int16
	Fd     uint32
	Buf    unsafe.Pointer
	pad3   uint32
	Nbytes uint64
	Offset int64
	pad4   int64
	Flags  uint32
	Resfd  uint32
}

type Event struct {
	Data unsafe.Pointer
	pad1 uint32
	Obj  *Iocb
	pad2 uint32
	Res  int64
	pad3 uint32
	Res2 int64
	pad4 uint32
}
