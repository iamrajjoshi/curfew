package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type File struct {
	Version  int                     `json:"version"`
	Sessions map[string]SessionState `json:"sessions"`
}

type SessionState struct {
	ForcedActive bool   `json:"forced_active,omitempty"`
	Disabled     bool   `json:"disabled,omitempty"`
	SnoozesUsed  int    `json:"snoozes_used,omitempty"`
	SnoozedUntil string `json:"snoozed_until,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type Manager struct {
	runtimePath string
	lockPath    string
}

func New(runtimePath string, lockPath string) *Manager {
	return &Manager{
		runtimePath: runtimePath,
		lockPath:    lockPath,
	}
}

func (m *Manager) Read() (File, error) {
	return m.withLock(func() (File, error) {
		return m.loadLocked()
	})
}

func (m *Manager) Update(now time.Time, fn func(*File) error) (File, error) {
	return m.withLock(func() (File, error) {
		state, err := m.loadLocked()
		if err != nil {
			return File{}, err
		}
		if err := fn(&state); err != nil {
			return File{}, err
		}
		if state.Version == 0 {
			state.Version = 1
		}
		if state.Sessions == nil {
			state.Sessions = map[string]SessionState{}
		}
		pruneExpired(&state, now)
		if err := m.writeLocked(state); err != nil {
			return File{}, err
		}
		return state, nil
	})
}

func (m *Manager) withLock(fn func() (File, error)) (File, error) {
	if err := os.MkdirAll(filepath.Dir(m.lockPath), 0o755); err != nil {
		return File{}, err
	}
	lockFile, err := os.OpenFile(m.lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return File{}, err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return File{}, err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	return fn()
}

func (m *Manager) loadLocked() (File, error) {
	bytes, err := os.ReadFile(m.runtimePath)
	if errors.Is(err, os.ErrNotExist) {
		return File{
			Version:  1,
			Sessions: map[string]SessionState{},
		}, nil
	}
	if err != nil {
		return File{}, err
	}

	var state File
	if err := json.Unmarshal(bytes, &state); err != nil {
		backup := fmt.Sprintf("%s.corrupt-%d", m.runtimePath, time.Now().Unix())
		_ = os.Rename(m.runtimePath, backup)
		return File{
			Version:  1,
			Sessions: map[string]SessionState{},
		}, nil
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionState{}
	}
	return state, nil
}

func (m *Manager) writeLocked(state File) error {
	if err := os.MkdirAll(filepath.Dir(m.runtimePath), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tempPath := fmt.Sprintf("%s.tmp.%d", m.runtimePath, os.Getpid())
	if err := os.WriteFile(tempPath, bytes, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, m.runtimePath)
}

func ParseTimestamp(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return timestamp, true
}

func pruneExpired(state *File, now time.Time) {
	threshold := now.Add(-72 * time.Hour)
	for date, session := range state.Sessions {
		if updatedAt, ok := ParseTimestamp(session.UpdatedAt); ok && updatedAt.Before(threshold) {
			delete(state.Sessions, date)
		}
	}
}
