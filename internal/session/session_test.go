package session

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", mgr.Count())
	}
}

func TestCreateSession(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	session, err := mgr.Create(
		"testuser",
		"testuser-cn",
		"192.0.2.1",
		"12345",
		"/tmp/acf",
		"/tmp/apf",
		"/tmp/arf",
	)

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session.ID == "" {
		t.Error("session ID is empty")
	}

	if len(session.ID) != 64 {
		t.Errorf("session ID length = %d, want 64", len(session.ID))
	}

	if session.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", session.Username)
	}

	if session.UntrustedIP != "192.0.2.1" {
		t.Errorf("UntrustedIP = %s, want 192.0.2.1", session.UntrustedIP)
	}

	if mgr.Count() != 1 {
		t.Errorf("expected 1 session, got %d", mgr.Count())
	}
}

func TestGetSession(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	created, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get existing session
	retrieved, err := mgr.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("retrieved session ID = %s, want %s", retrieved.ID, created.ID)
	}

	// Get non-existent session
	_, err = mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get should fail for non-existent session")
	}
}

func TestUpdateOIDCFlow(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	session, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update with OIDC flow data
	err = mgr.UpdateOIDCFlow(session.ID, "state123", "verifier456", "https://example.com/auth")
	if err != nil {
		t.Fatalf("UpdateOIDCFlow failed: %v", err)
	}

	// Verify update
	retrieved, err := mgr.Get(session.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.State != "state123" {
		t.Errorf("State = %s, want state123", retrieved.State)
	}

	if retrieved.CodeVerifier != "verifier456" {
		t.Errorf("CodeVerifier = %s, want verifier456", retrieved.CodeVerifier)
	}

	if retrieved.AuthURL != "https://example.com/auth" {
		t.Errorf("AuthURL = %s, want https://example.com/auth", retrieved.AuthURL)
	}
}

func TestGetByState(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	session, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = mgr.UpdateOIDCFlow(session.ID, "state123", "verifier456", "https://example.com/auth")
	if err != nil {
		t.Fatalf("UpdateOIDCFlow failed: %v", err)
	}

	// Get by state
	retrieved, err := mgr.GetByState("state123")
	if err != nil {
		t.Fatalf("GetByState failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("retrieved session ID = %s, want %s", retrieved.ID, session.ID)
	}

	// Get by non-existent state
	_, err = mgr.GetByState("nonexistent")
	if err == nil {
		t.Error("GetByState should fail for non-existent state")
	}
}

func TestDeleteSession(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	session, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = mgr.UpdateOIDCFlow(session.ID, "state123", "verifier456", "https://example.com/auth")
	if err != nil {
		t.Fatalf("UpdateOIDCFlow failed: %v", err)
	}

	if mgr.Count() != 1 {
		t.Errorf("expected 1 session, got %d", mgr.Count())
	}

	// Delete session
	mgr.Delete(session.ID)

	if mgr.Count() != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", mgr.Count())
	}

	// Verify session is gone
	_, err = mgr.Get(session.ID)
	if err == nil {
		t.Error("Get should fail after delete")
	}

	// Verify state index is cleared
	_, err = mgr.GetByState("state123")
	if err == nil {
		t.Error("GetByState should fail after delete")
	}

	// Delete non-existent session (should not panic)
	mgr.Delete("nonexistent")
}

func TestMarkResultWritten(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	sess, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if ok := mgr.MarkResultWritten(sess.ID); !ok {
		t.Fatal("expected MarkResultWritten to return true")
	}

	// Second call should return false (already marked)
	if ok := mgr.MarkResultWritten(sess.ID); ok {
		t.Fatal("expected MarkResultWritten to return false on second call")
	}

	// Unknown session should return false
	if ok := mgr.MarkResultWritten("nonexistent"); ok {
		t.Fatal("expected MarkResultWritten to return false for unknown session")
	}

	retrieved, err := mgr.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !retrieved.ResultWritten {
		t.Fatal("expected session to be marked ResultWritten")
	}
}

func TestResultWritten(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	sess, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if written, ok := mgr.ResultWritten(sess.ID); !ok || written {
		t.Fatalf("ResultWritten() = (%v,%v), want (false,true)", written, ok)
	}

	if ok := mgr.MarkResultWritten(sess.ID); !ok {
		t.Fatal("expected MarkResultWritten to return true")
	}

	if written, ok := mgr.ResultWritten(sess.ID); !ok || !written {
		t.Fatalf("ResultWritten() = (%v,%v), want (true,true)", written, ok)
	}

	mgr.Delete(sess.ID)
	if _, ok := mgr.ResultWritten(sess.ID); ok {
		t.Fatal("expected ResultWritten() ok=false after delete")
	}
}

func TestSessionExpiry(t *testing.T) {
	mgr := NewManager(100 * time.Millisecond)
	defer mgr.Stop()

	session, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Session should exist immediately
	_, err = mgr.Get(session.ID)
	if err != nil {
		t.Errorf("Get should succeed immediately: %v", err)
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Session should be expired
	_, err = mgr.Get(session.ID)
	if err == nil {
		t.Error("Get should fail for expired session")
	}
}

func TestCleanup(t *testing.T) {
	mgr := NewManager(100 * time.Millisecond)
	defer mgr.Stop()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		_, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	if mgr.Count() != 5 {
		t.Errorf("expected 5 sessions, got %d", mgr.Count())
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Trigger cleanup manually
	mgr.cleanup()

	if mgr.Count() != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", mgr.Count())
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewManager(5 * time.Minute)
	defer mgr.Stop()

	// Create sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := mgr.Create("testuser", "cn", "192.0.2.1", "12345", "/tmp/acf", "/tmp/apf", "/tmp/arf")
			if err != nil {
				t.Errorf("Create failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all creates
	for i := 0; i < 10; i++ {
		<-done
	}

	if mgr.Count() != 10 {
		t.Errorf("expected 10 sessions, got %d", mgr.Count())
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple session IDs and ensure they're unique
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID failed: %v", err)
		}

		if len(id) != 64 {
			t.Errorf("session ID length = %d, want 64", len(id))
		}

		if seen[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}

		seen[id] = true
	}
}
