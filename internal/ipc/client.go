package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// dialTimeout bounds both the connect and, absent a context deadline, the
// request/response exchange.
const dialTimeout = 5 * time.Second

// SendAuthRequest sends an authentication request to the daemon over the Unix
// socket at socketPath and waits for the response.
func SendAuthRequest(ctx context.Context, socketPath string, req *AuthRequest) (*AuthResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set overall deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(dialTimeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp AuthResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}
