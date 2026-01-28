package daemon

import (
	"os"
	"path/filepath"
)

const (
	baseDirName = ".tabs"
)

func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, baseDirName), nil
}

func EnsureBaseDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return base, nil
}

func SocketPath(baseDir string) string {
	return filepath.Join(baseDir, "daemon.sock")
}

func PIDPath(baseDir string) string {
	return filepath.Join(baseDir, "daemon.pid")
}

func LockPath(baseDir string) string {
	return filepath.Join(baseDir, "daemon.lock")
}

func LogPath(baseDir string) string {
	return filepath.Join(baseDir, "daemon.log")
}
