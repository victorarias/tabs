package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquirePIDCreatesAndReleases(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	lock, err := AcquirePID(baseDir)
	if err != nil {
		t.Fatalf("AcquirePID failed: %v", err)
	}
	if _, err := os.Stat(PIDPath(baseDir)); err != nil {
		t.Fatalf("pid file missing: %v", err)
	}
	if _, err := os.Stat(LockPath(baseDir)); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	if _, err := os.Stat(PIDPath(baseDir)); !os.IsNotExist(err) {
		t.Fatalf("pid file not removed")
	}
	if _, err := os.Stat(LockPath(baseDir)); !os.IsNotExist(err) {
		t.Fatalf("lock file not removed")
	}
}

func TestAcquirePIDDropsStaleLock(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	lockPath := LockPath(baseDir)
	if err := os.WriteFile(lockPath, []byte("lock"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	lock, err := AcquirePID(baseDir)
	if err != nil {
		t.Fatalf("AcquirePID failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release failed: %v", err)
	}
}

func TestAcquirePIDDetectsRunningProcess(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	pid := os.Getpid()
	pidPath := filepath.Join(baseDir, "daemon.pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if _, err := AcquirePID(baseDir); err == nil {
		t.Fatalf("expected error for running pid")
	}
}
