package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type PIDLock struct {
	pidPath  string
	lockPath string
}

func AcquirePID(baseDir string) (*PIDLock, error) {
	pidPath := PIDPath(baseDir)
	lockPath := LockPath(baseDir)

	if err := cleanupStalePID(pidPath, lockPath); err != nil {
		return nil, err
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("acquire daemon lock: %w", err)
	}
	_ = lockFile.Close()

	pid := os.Getpid()
	pidData := []byte(fmt.Sprintf("%d\n", pid))
	if err := os.WriteFile(pidPath, pidData, 0o600); err != nil {
		_ = os.Remove(lockPath)
		return nil, fmt.Errorf("write pid file: %w", err)
	}

	return &PIDLock{pidPath: pidPath, lockPath: lockPath}, nil
}

func (p *PIDLock) Release() error {
	var errs []string
	if p == nil {
		return nil
	}
	if err := os.Remove(p.pidPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err.Error())
	}
	if err := os.Remove(p.lockPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func cleanupStalePID(pidPath, lockPath string) error {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.Remove(lockPath)
			return nil
		}
		return fmt.Errorf("read pid file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		_ = os.Remove(pidPath)
		return nil
	}

	if processAlive(pid) {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	_ = os.Remove(pidPath)
	_ = os.Remove(lockPath)
	return nil
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}

func PIDFromFile(baseDir string) (int, error) {
	pidPath := filepath.Join(baseDir, "daemon.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, err
	}
	return pid, nil
}
