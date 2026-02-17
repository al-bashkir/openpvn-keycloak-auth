package ipc_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/al-bashkir/openvpn-keycloak-auth/internal/ipc"
)

// ExampleServer demonstrates how to create and use the IPC server (daemon side)
func ExampleServer() {
	// Create temp directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "auth.sock")

	// Define handler for auth requests
	handler := func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		// In a real implementation, this would:
		// 1. Create a session
		// 2. Generate OIDC authorization URL
		// 3. Write to auth_pending_file
		// 4. Return session ID and auth URL

		fmt.Printf("Received auth request for user: %s\n", req.Username)

		return &ipc.AuthResponse{
			Status:    ipc.StatusDeferred,
			SessionID: "example-session-123",
			AuthURL:   "https://keycloak.example.com/auth?session=123",
		}, nil
	}

	// Create and start server
	server := ipc.NewServer(socketPath, handler)
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = server.Stop() }()

	fmt.Println("IPC server running")

	// Keep running for a bit
	time.Sleep(100 * time.Millisecond)

	// Output:
	// IPC server running
}

// ExampleClient demonstrates how to use the IPC client (auth script side)
func ExampleClient() {
	// Create temp directory and server for demonstration
	tmpDir, err := os.MkdirTemp("", "ipc-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "auth.sock")

	// Start a test server
	handler := func(ctx context.Context, req *ipc.AuthRequest) (*ipc.AuthResponse, error) {
		return &ipc.AuthResponse{
			Status:    ipc.StatusDeferred,
			SessionID: "session-123",
			AuthURL:   "https://keycloak.example.com/auth",
		}, nil
	}

	server := ipc.NewServer(socketPath, handler)
	if err := server.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = server.Stop() }()

	time.Sleep(100 * time.Millisecond)

	// Create client
	client := ipc.NewClient(socketPath)

	// Create auth request (this would come from OpenVPN environment)
	req := &ipc.AuthRequest{
		Username:             "john.doe",
		CommonName:           "john.doe",
		UntrustedIP:          "192.0.2.1",
		UntrustedPort:        "12345",
		AuthControlFile:      "/tmp/auth_control",
		AuthPendingFile:      "/tmp/auth_pending",
		AuthFailedReasonFile: "/tmp/auth_failed",
	}

	// Send request
	resp, err := client.SendAuthRequest(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	// Handle response
	switch resp.Status {
	case ipc.StatusDeferred:
		fmt.Printf("Authentication deferred, session: %s\n", resp.SessionID)
		// In the real auth script, we would:
		// 1. Return exit code 2 (deferred)
		// 2. Daemon will write auth_pending_file with the auth URL
		// 3. Client opens browser
		// 4. Daemon eventually writes auth_control_file
	case ipc.StatusError:
		fmt.Printf("Authentication error: %s\n", resp.Error)
	}

	// Output:
	// Authentication deferred, session: session-123
}
