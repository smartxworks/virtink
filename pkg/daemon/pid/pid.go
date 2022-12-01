package pid

import (
	"fmt"
	"net"
	"syscall"
)

func GetPIDBySocket(socket string) (int, error) {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return 0, fmt.Errorf("dial socket: %s", err)
	}
	defer conn.Close()

	f, err := conn.(*net.UnixConn).File()
	if err != nil {
		return 0, fmt.Errorf("get connection file: %s", err)
	}
	defer f.Close()

	ucred, err := syscall.GetsockoptUcred(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return 0, fmt.Errorf("get ucred: %s", err)
	}
	return int(ucred.Pid), nil
}
