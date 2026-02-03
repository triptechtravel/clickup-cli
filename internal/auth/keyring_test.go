package auth

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/zalando/go-keyring"
)

func setConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLICKUP_CONFIG_DIR", dir)
	return dir
}

func TestStoreAndGetToken(t *testing.T) {
	keyring.MockInit()

	token := "pk_test_abc123"
	if err := StoreToken(token, "token"); err != nil {
		t.Fatalf("StoreToken() error: %v", err)
	}

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error: %v", err)
	}
	if got != token {
		t.Errorf("GetToken() = %q, want %q", got, token)
	}
}

func TestStoreAndGetAuthMethod(t *testing.T) {
	keyring.MockInit()

	if err := StoreToken("pk_test_abc123", "oauth"); err != nil {
		t.Fatalf("StoreToken() error: %v", err)
	}

	got := GetAuthMethod()
	if got != "oauth" {
		t.Errorf("GetAuthMethod() = %q, want %q", got, "oauth")
	}
}

func TestClearToken(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	if err := StoreToken("pk_test_abc123", "token"); err != nil {
		t.Fatalf("StoreToken() error: %v", err)
	}

	if err := ClearToken(); err != nil {
		t.Fatalf("ClearToken() error: %v", err)
	}

	_, err := GetToken()
	if err == nil {
		t.Fatal("GetToken() after ClearToken() should return an error")
	}
}

func TestGetToken_NoCredentials(t *testing.T) {
	keyring.MockInit()
	setConfigDir(t)

	_, err := GetToken()
	if err == nil {
		t.Fatal("GetToken() with no credentials should return an error")
	}
}

func TestKeyringFallbackToFile(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	// Capture stdout to suppress warning output during test.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	token := "pk_file_fallback"
	if err := StoreToken(token, "token"); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	// Drain the pipe so it doesn't block.
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify the auth file was created.
	if _, err := os.Stat(config.AuthFile()); err != nil {
		t.Fatalf("auth file not created: %v", err)
	}

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() from file fallback error: %v", err)
	}
	if got != token {
		t.Errorf("GetToken() = %q, want %q", got, token)
	}
}

func TestKeyringFallbackWarning(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := StoreToken("pk_test", "token"); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "plain text") {
		t.Errorf("expected warning about plain text storage, got: %q", output)
	}
}

func TestClearTokenFileFallback(t *testing.T) {
	keyring.MockInitWithError(fmt.Errorf("keyring unavailable"))
	setConfigDir(t)

	// Capture stdout to suppress warning.
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	if err := StoreToken("pk_test", "token"); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("StoreToken() error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	if err := ClearToken(); err != nil {
		t.Fatalf("ClearToken() error: %v", err)
	}

	_, err := GetToken()
	if err == nil {
		t.Fatal("GetToken() after ClearToken() should return an error")
	}
}
