package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// AuthRequestHandler is the function type for handling auth requests
type AuthRequestHandler func(ctx context.Context, req *AuthRequest) (*AuthResponse, error)

// Server is the IPC server that listens on a Unix socket for auth requests
type Server struct {
	socketPath string
	listener   net.Listener
	handler    AuthRequestHandler
	wg         sync.WaitGroup
	stopChan   chan struct{}
	mu         sync.Mutex
}

// NewServer creates a new IPC server
func NewServer(socketPath string, handler AuthRequestHandler) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the IPC server
func (s *Server) Start(ctx context.Context) error {
	// Ensure the directory exists.
	// Use 0755 so any local process can traverse the directory.
	// Access control is enforced at the socket level.
	dir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove old socket if it exists
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old socket: %w", err)
	}

	// Create Unix listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Set socket permissions: 0660 (owner + group read/write).
	// The daemon should run in the same group as OpenVPN (e.g., openvpn)
	// so that the auth script can connect. World access is denied to
	// prevent untrusted local users from submitting forged auth requests.
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	slog.Info("IPC server started", "socket", s.socketPath)

	// Start accept loop in goroutine
	s.wg.Add(1)
	go s.acceptLoop(ctx)

	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				// Server is stopping, this is expected
				return
			default:
				slog.Error("failed to accept connection", "error", err)
				continue
			}
		}

		// Handle connection in goroutine
		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single IPC connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer func() { _ = conn.Close() }()

	// Decode request
	var req AuthRequest
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&req); err != nil {
		slog.Error("failed to decode request", "error", err)
		s.sendErrorResponse(conn, "invalid request format")
		return
	}

	// Validate request type
	if req.Type != MessageTypeAuthRequest {
		slog.Error("invalid request type", "type", req.Type)
		s.sendErrorResponse(conn, "invalid request type")
		return
	}

	slog.Info("auth request received",
		"username", req.Username,
		"ip", req.UntrustedIP,
		"common_name", req.CommonName,
	)

	// Call handler
	resp, err := s.handler(ctx, &req)
	if err != nil {
		slog.Error("handler error", "error", err)
		s.sendErrorResponse(conn, err.Error())
		return
	}

	// Send response
	resp.Type = MessageTypeAuthResponse
	enc := json.NewEncoder(conn)
	if err := enc.Encode(resp); err != nil {
		slog.Error("failed to send response", "error", err)
		return
	}

	slog.Debug("auth response sent", "status", resp.Status, "session_id", resp.SessionID)
}

// sendErrorResponse sends an error response to the client
func (s *Server) sendErrorResponse(conn net.Conn, errMsg string) {
	resp := &AuthResponse{
		Type:   MessageTypeAuthResponse,
		Status: StatusError,
		Error:  errMsg,
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(resp); err != nil {
		slog.Error("failed to send error response", "error", err)
	}
}

// Stop stops the IPC server gracefully
func (s *Server) Stop() error {
	slog.Info("stopping IPC server")

	// Signal accept loop to stop
	close(s.stopChan)

	// Close listener
	s.mu.Lock()
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			slog.Warn("failed to close listener", "error", err)
		}
	}
	s.mu.Unlock()

	// Wait for all connections to finish
	s.wg.Wait()

	// Remove socket file
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove socket file", "error", err)
	}

	slog.Info("IPC server stopped")
	return nil
}
