package runtime

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestUpdateIsSafeAcrossConcurrentWriters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager := New(filepath.Join(dir, "runtime.json"), filepath.Join(dir, "runtime.lock"))
	now := time.Date(2026, 4, 24, 23, 0, 0, 0, time.UTC)

	const writers = 16
	var wait sync.WaitGroup
	wait.Add(writers)
	for i := 0; i < writers; i++ {
		go func() {
			defer wait.Done()
			_, err := manager.Update(now, func(file *File) error {
				state := file.Sessions["2026-04-24"]
				state.SnoozesUsed++
				state.UpdatedAt = now.Format(time.RFC3339)
				file.Sessions["2026-04-24"] = state
				return nil
			})
			if err != nil {
				t.Errorf("update: %v", err)
			}
		}()
	}
	wait.Wait()

	state, err := manager.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := state.Sessions["2026-04-24"].SnoozesUsed; got != writers {
		t.Fatalf("snoozes_used = %d, want %d", got, writers)
	}
}

func TestReadRecoversFromCorruptState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "runtime.json")
	if err := os.WriteFile(runtimePath, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	manager := New(runtimePath, filepath.Join(dir, "runtime.lock"))
	state, err := manager.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if state.Version != 1 {
		t.Fatalf("version = %d, want 1", state.Version)
	}
	if len(state.Sessions) != 0 {
		t.Fatalf("sessions = %d, want 0", len(state.Sessions))
	}

	matches, err := filepath.Glob(runtimePath + ".corrupt-*")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("corrupt backups = %d, want 1", len(matches))
	}
}
