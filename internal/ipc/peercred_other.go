//go:build !linux

package ipc

import "net"

func validatePeerCredentials(conn net.Conn) error {
	return nil
}
