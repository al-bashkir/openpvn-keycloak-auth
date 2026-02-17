package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client is the IPC client used by the auth script to communicate with the daemon
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new IPC client
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

// SendAuthRequest sends an authentication request to the daemon and waits for response
func (c *Client) SendAuthRequest(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	// Set request type
	req.Type = MessageTypeAuthRequest

	// Connect to Unix socket with timeout
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set overall deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	// Send request
	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var resp AuthResponse
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Validate response type
	if resp.Type != MessageTypeAuthResponse {
		return nil, fmt.Errorf("invalid response type: %s", resp.Type)
	}

	return &resp, nil
}

// SetTimeout sets the connection timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}
