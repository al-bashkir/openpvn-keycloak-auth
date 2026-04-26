//go:build linux

package ipc

import (
	"fmt"
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
		cred, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
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
	if cred.Uid == uint32(os.Geteuid()) || cred.Uid == 0 {
		return nil
	}

	return fmt.Errorf("peer uid %d is not allowed", cred.Uid)
}
