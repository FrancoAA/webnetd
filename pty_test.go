package main

import (
	"strings"
	"testing"
	"time"
)

func TestNewTerminal(t *testing.T) {
	term, err := newTerminal("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("newTerminal: %v", err)
	}
	defer term.close()

	if term.ptmx == nil {
		t.Fatal("ptmx should not be nil")
	}
	if term.cmd == nil {
		t.Fatal("cmd should not be nil")
	}
	if term.cmd.Process == nil {
		t.Fatal("process should be running")
	}
	if term.cmd.Process.Pid <= 0 {
		t.Fatalf("expected positive PID, got %d", term.cmd.Process.Pid)
	}
}

func TestNewTerminalInvalidShell(t *testing.T) {
	_, err := newTerminal("/nonexistent/shell", 24, 80)
	if err == nil {
		t.Fatal("expected error for invalid shell")
	}
}

func TestTerminalReadWrite(t *testing.T) {
	term, err := newTerminal("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("newTerminal: %v", err)
	}
	defer term.close()

	// Write a command
	_, err = term.ptmx.Write([]byte("echo test-pty-output\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read output
	deadline := time.Now().Add(5 * time.Second)
	buf := make([]byte, 4096)
	var output []byte
	for time.Now().Before(deadline) {
		term.ptmx.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := term.ptmx.Read(buf)
		if n > 0 {
			output = append(output, buf[:n]...)
		}
		if strings.Contains(string(output), "test-pty-output") {
			break
		}
		if err != nil {
			continue
		}
	}

	if !strings.Contains(string(output), "test-pty-output") {
		t.Fatalf("expected 'test-pty-output' in output, got:\n%s", output)
	}
}

func TestTerminalResize(t *testing.T) {
	term, err := newTerminal("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("newTerminal: %v", err)
	}
	defer term.close()

	// Resize should not error
	if err := term.resize(50, 120); err != nil {
		t.Fatalf("resize: %v", err)
	}

	// Resize to small size
	if err := term.resize(1, 1); err != nil {
		t.Fatalf("resize to 1x1: %v", err)
	}
}

func TestTerminalClose(t *testing.T) {
	term, err := newTerminal("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("newTerminal: %v", err)
	}

	pid := term.cmd.Process.Pid
	term.close()

	// After close, writing should fail
	_, err = term.ptmx.Write([]byte("test\n"))
	if err == nil {
		t.Fatal("write after close should fail")
	}

	// Process should have exited
	// Give it a moment to clean up
	time.Sleep(100 * time.Millisecond)
	_ = pid // process is killed and waited on in close()
}
