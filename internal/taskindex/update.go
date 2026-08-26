package taskindex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Update applies a change to a workspace's index under an advisory lock,
// re-reading the current on-disk state first.
//
// Without this, two concurrent searches each load the index, sync into their
// own copy, and write it back — so the last one to finish silently discards the
// other's work. That is not merely wasted effort: a truncated sync stamps the
// reconcile clock, so a process that gave up early can overwrite a complete
// rebuild and pin the degraded result for a week, and a concurrent write can
// drop the RequestFullSync that a stranded watermark depends on to recover.
//
// The lock covers only the read-modify-write, never the sync itself. Holding it
// across a sync would make one search block another for up to a minute; holding
// it across a file rewrite costs milliseconds.
//
// apply reports whether it changed anything; when it returns false nothing is
// written, which is what keeps a no-change search from rewriting a
// multi-megabyte file.
func Update(dir, workspace string, apply func(*Index) bool) error {
	release, err := acquireLock(dir)
	if err != nil {
		return err
	}
	defer release()

	idx, err := Load(dir, workspace)
	if err != nil {
		return err
	}
	if !apply(idx) {
		return nil
	}
	return Save(dir, idx)
}

const (
	// lockTimeout bounds how long we wait for another process's read-modify-write.
	// That critical section is milliseconds, so anything near this means a stuck
	// or dead holder.
	lockTimeout = 5 * time.Second
	// lockStale is when a lock file is assumed abandoned — a process killed
	// between creating it and removing it would otherwise block every future
	// search forever.
	lockStale = 30 * time.Second
	lockPoll  = 10 * time.Millisecond
)

func lockPath(dir string) string { return filepath.Join(dir, "index.lock") }

// acquireLock takes the advisory lock, returning a release function.
//
// A lock that cannot be taken is not fatal: the caller degrades to a possible
// lost update, which is strictly better than refusing to search.
func acquireLock(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	path := lockPath(dir)
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire index lock: %w", err)
		}

		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > lockStale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			// Proceed unlocked rather than fail the search.
			return func() {}, nil
		}
		time.Sleep(lockPoll)
	}
}
