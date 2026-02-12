// Package drivecache manages a local cache of Windows drive information.
// The cache is stored at ~/.cache/wstart/drives.json and refreshed by
// invoking the Windows helper binary with --drives.
package drivecache

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sverrirab/wsl-host-start/internal/protocol"
)

const defaultTTL = 1 * time.Hour

// Cache manages the drive info cache file.
type Cache struct {
	path string
	ttl  time.Duration
}

type cacheFile struct {
	Timestamp time.Time               `json:"timestamp"`
	Data      protocol.DrivesResponse `json:"data"`
}

// New creates a cache manager. If ttl is 0, defaultTTL is used.
func New(ttl time.Duration) *Cache {
	if ttl == 0 {
		ttl = defaultTTL
	}
	return &Cache{
		path: cachePath(),
		ttl:  ttl,
	}
}

// Get returns cached drive info, refreshing if stale or missing.
func (c *Cache) Get(helperPath string) (*protocol.DrivesResponse, error) {
	if data, err := c.load(); err == nil {
		return data, nil
	}
	return c.Refresh(helperPath)
}

// Refresh invokes the helper to enumerate drives and updates the cache.
func (c *Cache) Refresh(helperPath string) (*protocol.DrivesResponse, error) {
	cmd := exec.Command(helperPath, "--drives")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running %s --drives: %w", helperPath, err)
	}

	var resp protocol.DrivesResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing drive info: %w", err)
	}

	if err := c.save(&resp); err != nil {
		// Non-fatal: cache write failure shouldn't block operation.
		fmt.Fprintf(os.Stderr, "wstart: warning: could not write drive cache: %v\n", err)
	}

	return &resp, nil
}

func (c *Cache) load() (*protocol.DrivesResponse, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, err
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, err
	}

	if time.Since(cf.Timestamp) > c.ttl {
		return nil, fmt.Errorf("cache expired")
	}

	return &cf.Data, nil
}

func (c *Cache) save(resp *protocol.DrivesResponse) error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	cf := cacheFile{
		Timestamp: time.Now(),
		Data:      *resp,
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}

func cachePath() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "wstart", "drives.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "wstart", "drives.json")
}
