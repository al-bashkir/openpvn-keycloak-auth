//go:build linux

package ipc

import (
	"fmt"
	"math"
	"net"
	"os"
	"syscall"
)

func validatePeerCredentials(conn net.Conn) error {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("expected Unix connection, got %T", conn)
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("failed to access Unix connection: %w", err)
	}

	var cred *syscall.Ucred
	var controlErr error
	if err := rawConn.Control(func(fd uintptr) {
		socketFD, err := socketFileDescriptor(fd)
		if err != nil {
			controlErr = err
			return
		}
		cred, controlErr = syscall.GetsockoptUcred(socketFD, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return fmt.Errorf("failed to read peer credentials: %w", err)
	}
	if controlErr != nil {
		return fmt.Errorf("failed to read peer credentials: %w", controlErr)
	}
	if cred == nil {
		return fmt.Errorf("peer credentials unavailable")
	}

	// The auth helper should run as the same service user as the daemon after
	// OpenVPN drops privileges. Root is also allowed for manual/admin tests.
	currentUID, err := currentEffectiveUID()
	if err != nil {
		return err
	}
	if cred.Uid == currentUID || cred.Uid == 0 {
		return nil
	}

	return fmt.Errorf("peer uid %d is not allowed", cred.Uid)
}

func socketFileDescriptor(fd uintptr) (int, error) {
	if fd > uintptr(math.MaxInt) {
		return 0, fmt.Errorf("socket file descriptor %d exceeds max int", fd)
	}
	return int(fd), nil
}

func currentEffectiveUID() (uint32, error) {
	euid := os.Geteuid()
	if euid < 0 || euid > math.MaxUint32 {
		return 0, fmt.Errorf("effective uid %d exceeds uint32 range", euid)
	}
	return uint32(euid), nil
}
