package tree

import (
	"os"
	"path/filepath"
)

// Root resolves the volume root (SPEC §1):
//
//  1. the app's own env var (e.g. "PENSUM_ROOT"), if appEnv is non-empty and set
//  2. PANTHEON_ROOT
//  3. the caller-supplied fallback
//  4. ~/vol_f
func Root(appEnv, fallback string) string {
	if appEnv != "" {
		if v := os.Getenv(appEnv); v != "" {
			return v
		}
	}
	if v := os.Getenv("PANTHEON_ROOT"); v != "" {
		return v
	}
	if fallback != "" {
		return fallback
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "vol_f" // last resort: relative to cwd
	}
	return filepath.Join(home, "vol_f")
}
