package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chaitin/chaitin-cli/products/dsensor/internal/client"
)

// State is the cached state stored on disk.
type State struct {
	SpecHash      string    `json:"spec_hash"`
	ServerVersion string    `json:"server_version"`
	LastChecked   time.Time `json:"last_checked"`
}

// DefaultDir returns the default cache directory.
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dsensor/cache"
	}
	return filepath.Join(home, ".dsensor", "cache")
}

// cachePath returns the path to the state file.
func cachePath(dir string) string {
	return filepath.Join(dir, "state.json")
}

// Load reads the cached state from disk.
func Load(dir string) (*State, error) {
	path := cachePath(dir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("缓存文件损坏: %w", err)
	}
	return &state, nil
}

// Save writes the state to disk.
func Save(dir string, state *State) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	state.LastChecked = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	path := cachePath(dir)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入缓存失败: %w", err)
	}
	return nil
}

// SpecHash computes the SHA256 hash of the embedded spec bytes.
func SpecHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// CheckVersion compares cached version with server version.
// Returns true if the cache is valid, false if it needs refresh.
func CheckVersion(cached *State, cl *client.Client) (*State, bool) {
	v, err := cl.GetVersion()
	if err != nil {
		if cached != nil {
			fmt.Fprintf(os.Stderr, "WARNING: 无法连接服务器获取版本信息，降级使用缓存\n")
			return cached, true
		}
		return nil, false
	}

	newState := &State{
		SpecHash:      cached.SpecHash,
		ServerVersion: v.ManagerVersion,
	}

	if cached == nil {
		return newState, false
	}

	if cached.ServerVersion != v.ManagerVersion {
		fmt.Fprintf(os.Stderr, "INFO: 服务器版本已更新: %s -> %s\n", cached.ServerVersion, v.ManagerVersion)
		return newState, false
	}

	return cached, true
}
