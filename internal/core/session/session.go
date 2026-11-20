package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SessionManager manages active game context per terminal/shell session
type SessionManager struct {
	BaseDir string
}

// NewSessionManager initializes a session manager targeting /run/user/<uid>/modr
func NewSessionManager() *SessionManager {
	uid := os.Getuid()
	runUser := fmt.Sprintf("/run/user/%d/modr", uid)
	if fi, err := os.Stat(fmt.Sprintf("/run/user/%d", uid)); err == nil && fi.IsDir() {
		_ = os.MkdirAll(runUser, 0700)
		return &SessionManager{BaseDir: runUser}
	}

	fallback := filepath.Join(os.TempDir(), fmt.Sprintf("modr-%d", uid))
	_ = os.MkdirAll(fallback, 0700)
	return &SessionManager{BaseDir: fallback}
}

// GetSessionKey determines the unique session identifier for the current terminal
func (s *SessionManager) GetSessionKey() string {
	// 1. Try reading the controlling TTY from stdin (/proc/self/fd/0)
	if target, err := os.Readlink("/proc/self/fd/0"); err == nil && strings.HasPrefix(target, "/dev/pts/") {
		ptsNum := strings.TrimPrefix(target, "/dev/pts/")
		return fmt.Sprintf("pts-%s", ptsNum)
	}

	// 2. Try session ID from /proc/self/stat
	if sid := getSessionID(); sid > 0 {
		return fmt.Sprintf("sid-%d", sid)
	}

	// 3. Fallback to parent process ID (shell PID)
	return fmt.Sprintf("ppid-%d", os.Getppid())
}

// getSessionID reads the session ID from /proc/self/stat on Linux
func getSessionID() int {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	str := string(data)
	lastParen := strings.LastIndex(str, ")")
	if lastParen == -1 || lastParen+2 >= len(str) {
		return 0
	}
	fields := strings.Fields(str[lastParen+2:])
	if len(fields) > 3 {
		var sid int
		if _, err := fmt.Sscanf(fields[3], "%d", &sid); err == nil && sid > 0 {
			return sid
		}
	}
	return 0
}

// GetActiveGame returns the currently active game for this session, if any
func (s *SessionManager) GetActiveGame() (string, error) {
	// Priority 1: Environment variable MODR_GAME
	if env := os.Getenv("MODR_GAME"); env != "" {
		return strings.TrimSpace(env), nil
	}

	// Priority 2: Session file
	key := s.GetSessionKey()
	sessionFile := filepath.Join(s.BaseDir, key+".context")
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	game := strings.TrimSpace(string(data))
	return game, nil
}

// SetActiveGame records the active game for this session
func (s *SessionManager) SetActiveGame(gameName string) error {
	key := s.GetSessionKey()
	sessionFile := filepath.Join(s.BaseDir, key+".context")
	return os.WriteFile(sessionFile, []byte(strings.TrimSpace(gameName)+"\n"), 0600)
}

// ClearActiveGame unsets the active game for this session
func (s *SessionManager) ClearActiveGame() error {
	key := s.GetSessionKey()
	sessionFile := filepath.Join(s.BaseDir, key+".context")
	err := os.Remove(sessionFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
