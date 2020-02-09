// +build linux
// +build i386 arm mips

package ioengine

import (
	"unsafe"
)

type Iocb struct {
	data   unsafe.Pointer
	pad1   uint32
	key    uint32
	pad2   uint32
	opcode int16
	prio   int16
	fd     uint32
	buf    unsafe.Pointer
	pad3   uint32
	nbytes uint64
	offset int64
	pad4   int64
	flags  uint32
	resfd  uint32
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
