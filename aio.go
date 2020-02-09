// +build linux

package ioengine

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/josiahpwhite/fencer"
)

type IocbCmd int16

const (
	IOCmdPread IocbCmd = iota
	IOCmdPwrite
	IOCmdFSync
	IOCmdFDSync
	IOCmdPoll
	IOCmdNoop
	IOCmdPreadv
	IOCmdPwritev
)

type timespec struct {
	sec  int
	nsec int
}

type kernelAIORingHdr struct {
	id   uint32
	nr   uint32
	head uint32
	tail uint32

	magic            uint32
	compatFeatures   uint32
	incompatFeatures uint32
	headerLength     uint32

	//struct io_event events[0];
}

type IOContext uint

func NewIOContext(maxEvents int) (IOContext, error) {
	var ioctx IOContext
	_, _, err := syscall.Syscall(syscall.SYS_IO_SETUP, uintptr(maxEvents), uintptr(unsafe.Pointer(&ioctx)), 0)
	if err != 0 {
		return 0, os.NewSyscallError("IO_SETUP", err)
	}
	return ioctx, nil
}

func (ioctx IOContext) Destroy() error {
	_, _, err := syscall.Syscall(syscall.SYS_IO_DESTROY, uintptr(ioctx), 0, 0)
	if err != 0 {
		return os.NewSyscallError("IO_DESTROY", err)
	}
	return nil
}

func (ioctx IOContext) Submit(iocbs []*Iocb) (int, error) {
	var p unsafe.Pointer
	if len(iocbs) > 0 {
		p = unsafe.Pointer(&iocbs[0])
	} else {
		p = unsafe.Pointer(&zero)
	}
	n, _, err := syscall.Syscall(syscall.SYS_IO_SUBMIT, uintptr(ioctx), uintptr(len(iocbs)), uintptr(p))
	if err != 0 {
		return 0, os.NewSyscallError("IO_SUBMIT", err)
	}
	return int(n), nil
}

func (ioctx IOContext) Cancel(iocbs []Iocb, events []Event) (int, error) {
	var p0, p1 unsafe.Pointer
	if len(iocbs) > 0 {
		p0 = unsafe.Pointer(&iocbs[0])
	} else {
		p0 = unsafe.Pointer(&zero)
	}
	if len(events) > 0 {
		p1 = unsafe.Pointer(&events[0])
	} else {
		p1 = unsafe.Pointer(&zero)
	}
	n, _, err := syscall.Syscall(syscall.SYS_IO_CANCEL, uintptr(ioctx), uintptr(p0), uintptr(p1))
	if err != 0 {
		return 0, os.NewSyscallError("IO_CANCEL", err)
	}
	return int(n), nil
}

/* getEventsUserland will check for events in userland, returns the number of events
   and a boolean indication weather the event was handled.
*/
func (ioctx IOContext) getEventsUserland(minnr, nr int, events []Event, timeout timespec) (int, bool) {
	if ioctx == 0 {
		return 0, false
	}

	ring := ((*kernelAIORingHdr)(unsafe.Pointer(uintptr(ioctx))))

	// AIO_RING_MAGIC = 0xa10a10a1
	if ring.magic != 0xa10a10a1 {
		return 0, false
	}
	ring_events := (*[1 << 31]Event)(unsafe.Pointer(uintptr(ioctx) + unsafe.Sizeof(*ring)))[:ring.nr:ring.nr]

	i := 0
	for i < nr {
		head := ring.head
		if head == ring.tail {
			// There are no more completions
			break
		} else {
			// There is another completion to reap
			events[i] = ring_events[head]
			fencer.LFence()
			ring.head = (head + 1) % ring.nr
			i++
		}
	}

	if i == 0 && timeout.sec == 0 && timeout.nsec == 0 && ring.head == ring.tail {
		return 0, true
	}

	if i > 0 {
		return i, true
	}

	return 0, false
}

func (ioctx IOContext) GetEvents(minnr, nr int, events []Event, timeout timespec) (int, error) {
	if un, handled := ioctx.getEventsUserland(minnr, nr, events, timeout); handled {
		return int(un), nil
	}

	var p unsafe.Pointer
	if len(events) > 0 {
		p = unsafe.Pointer(&events[0])
	} else {
		p = unsafe.Pointer(&zero)
	}

	n, _, err := syscall.Syscall6(syscall.SYS_IO_GETEVENTS, uintptr(ioctx), uintptr(minnr),
		uintptr(nr), uintptr(p), uintptr(unsafe.Pointer(&timeout)), uintptr(0))
	if err != 0 {
		return 0, os.NewSyscallError("IO_GETEVENTS", err)
	}
	return int(n), nil
}

func NewIocb(fd uint32) *Iocb {
	return &Iocb{fd: fd, prio: 0}
}

func (iocb *Iocb) PrepPread(buf []byte, bufLen int, offset int64) {
	var p unsafe.Pointer
	if len(buf) > 0 {
		p = unsafe.Pointer(&buf[0])
	} else {
		p = unsafe.Pointer(&zero)
	}
	iocb.opcode = int16(IOCmdPread)
	iocb.buf = p
	iocb.nbytes = uint64(bufLen)
	iocb.offset = offset
}

func (iocb *Iocb) PrepPwrite(buf []byte, bufLen int, offset int64) {
	var p unsafe.Pointer
	if len(buf) > 0 {
		p = unsafe.Pointer(&buf[0])
	} else {
		p = unsafe.Pointer(&zero)
	}
	iocb.opcode = int16(IOCmdPwrite)
	iocb.buf = p
	iocb.nbytes = uint64(bufLen)
	iocb.offset = offset
}

func (iocb *Iocb) PrepPreadv(bs [][]byte, offset int64) {
	iovecs := bytes2Iovec(bs)
	var p unsafe.Pointer
	if len(iovecs) > 0 {
		p = unsafe.Pointer(&iovecs[0])
	} else {
		p = unsafe.Pointer(&zero)
	}
	iocb.opcode = int16(IOCmdPreadv)
	iocb.buf = p
	iocb.nbytes = uint64(len(iovecs))
	iocb.offset = offset
}

func (iocb *Iocb) PrepPwritev(bs [][]byte, offset int64) {
	iovecs := bytes2Iovec(bs)
	var p unsafe.Pointer
	if len(iovecs) > 0 {
		p = unsafe.Pointer(&iovecs[0])
	} else {
		p = unsafe.Pointer(&zero)
	}
	iocb.opcode = int16(IOCmdPwritev)
	iocb.buf = p
	iocb.nbytes = uint64(len(iovecs))
	iocb.offset = offset
}

func (iocb *Iocb) PrepFSync() {
	iocb.opcode = int16(IOCmdFSync)
}

func (iocb *Iocb) PrepFDSync() {
	iocb.opcode = int16(IOCmdFDSync)
}

func (iocb *Iocb) SetEventFd(eventfd int) {
	iocb.flags |= (1 << 0)
	iocb.resfd = uint32(eventfd)
}

func (iocb *Iocb) OpCode() IocbCmd {
	return IocbCmd(iocb.opcode)
}
