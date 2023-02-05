package ebpfev

import "time"

type EventData struct {
	Timestamp   time.Time
	ContainerID string
	Ppid        string
	Pid         string
	SyscallOp   string
	SyscallArgs []string
	Exe         string
	Cmd         string
}
