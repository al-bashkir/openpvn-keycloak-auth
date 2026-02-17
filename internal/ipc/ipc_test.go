package ipc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientServerCommunication(t *testing.T) {
	// Create temp directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create handler that returns a successful response
	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		return &AuthResponse{
			Status:    StatusDeferred,
			SessionID: "test-session-123",
			AuthURL:   "https://keycloak.example.com/auth?session=123",
		}, nil
	}

	// Create and start server
	server := NewServer(socketPath, handler)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	client := NewClient(socketPath)

	// Send request
	req := &AuthRequest{
		Username:             "testuser",
		CommonName:           "testuser",
		UntrustedIP:          "192.0.2.1",
		UntrustedPort:        "12345",
		AuthControlFile:      "/tmp/test_acf",
		AuthPendingFile:      "/tmp/test_apf",
		AuthFailedReasonFile: "/tmp/test_arf",
	}

	resp, err := client.SendAuthRequest(ctx, req)
	if err != nil {
		t.Fatalf("SendAuthRequest failed: %v", err)
	}

	// Validate response
	if resp.Type != MessageTypeAuthResponse {
		t.Errorf("expected type %s, got %s", MessageTypeAuthResponse, resp.Type)
	}
	if resp.Status != StatusDeferred {
		t.Errorf("expected status %s, got %s", StatusDeferred, resp.Status)
	}
	if resp.SessionID != "test-session-123" {
		t.Errorf("expected session_id test-session-123, got %s", resp.SessionID)
	}
	if resp.AuthURL == "" {
		t.Error("expected auth_url to be set")
	}
}

func TestServerHandlerError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create handler that returns an error
	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		return &AuthResponse{
			Status: StatusError,
			Error:  "daemon not initialized",
		}, nil
	}

	server := NewServer(socketPath, handler)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client := NewClient(socketPath)

	req := &AuthRequest{
		Username:        "testuser",
		AuthControlFile: "/tmp/test_acf",
		AuthPendingFile: "/tmp/test_apf",
	}

	resp, err := client.SendAuthRequest(ctx, req)
	if err != nil {
		t.Fatalf("SendAuthRequest failed: %v", err)
	}

	if resp.Status != StatusError {
		t.Errorf("expected status error, got %s", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected error message to be set")
	}
}

func TestClientConnectionFailure(t *testing.T) {
	// Try to connect to non-existent socket
	client := NewClient("/nonexistent/path/test.sock")

	req := &AuthRequest{
		Username: "testuser",
	}

	_, err := client.SendAuthRequest(context.Background(), req)
	if err == nil {
		t.Error("expected error when connecting to non-existent socket")
	}
}

func TestServerSocketPermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		return &AuthResponse{Status: StatusDeferred}, nil
	}

	server := NewServer(socketPath, handler)

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	// Check socket permissions
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("failed to stat socket: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0660) | os.ModeSocket

	if mode != expectedMode {
		t.Errorf("expected socket mode %v, got %v", expectedMode, mode)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Handler that takes a bit of time
	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		time.Sleep(200 * time.Millisecond)
		return &AuthResponse{Status: StatusDeferred}, nil
	}

	server := NewServer(socketPath, handler)

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Start a request in background
	go func() {
		client := NewClient(socketPath)
		req := &AuthRequest{Username: "testuser"}
		_, _ = client.SendAuthRequest(context.Background(), req)
	}()

	time.Sleep(50 * time.Millisecond)

	// Stop server - should wait for request to complete
	if err := server.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Socket should be removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("socket file should be removed after stop")
	}
}

func TestMultipleConcurrentRequests(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Handler that returns unique session ID based on username
	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		return &AuthResponse{
			Status:    StatusDeferred,
			SessionID: "session-" + req.Username,
		}, nil
	}

	server := NewServer(socketPath, handler)

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Send multiple concurrent requests
	numRequests := 10
	results := make(chan *AuthResponse, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(n int) {
			client := NewClient(socketPath)
			req := &AuthRequest{
				Username: string(rune('A' + n)),
			}

			resp, err := client.SendAuthRequest(context.Background(), req)
			if err != nil {
				errors <- err
				return
			}
			results <- resp
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-errors:
			t.Errorf("request failed: %v", err)
		case resp := <-results:
			if resp.Status != StatusDeferred {
				t.Errorf("expected status deferred, got %s", resp.Status)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for responses")
		}
	}
}

func TestClientTimeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Handler that sleeps longer than client timeout
	handler := func(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
		time.Sleep(2 * time.Second)
		return &AuthResponse{Status: StatusDeferred}, nil
	}

	server := NewServer(socketPath, handler)

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("server.Stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client := NewClient(socketPath)
	client.SetTimeout(500 * time.Millisecond)

	req := &AuthRequest{Username: "testuser"}

	_, err = client.SendAuthRequest(context.Background(), req)
	if err == nil {
		t.Error("expected timeout error")
	}
}
