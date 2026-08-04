// Package registry reads and writes Buzz Desktop's local managed-agent
// registry (managed-agents.json). It has no TUI dependency so it can be
// tested and reused independently of the interactive frontend.
package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// AppID is Buzz Desktop's Tauri app identifier.
// See desktop/src-tauri/tauri.conf.json:5 in the Buzz source tree.
const AppID = "xyz.block.buzz.app"

// Row is one entry in managed-agents.json. Deliberately untyped: this tool
// does not track Buzz's full schema, so unknown fields must survive a
// read-modify-write round trip unchanged.
type Row = map[string]any

// CandidateStores returns every path this tool knows to check for
// managed-agents.json, in priority order, across macOS/Windows/Linux and a
// couple of older identifier layouts.
func CandidateStores() []string {
	home, _ := os.UserHomeDir()
	var roots []string

	switch runtime.GOOS {
	case "darwin":
		roots = append(roots, filepath.Join(home, "Library", "Application Support", AppID))
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			roots = append(roots, filepath.Join(appdata, AppID))
		}
	default:
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			xdgData = filepath.Join(home, ".local", "share")
		}
		xdgConf := os.Getenv("XDG_CONFIG_HOME")
		if xdgConf == "" {
			xdgConf = filepath.Join(home, ".config")
		}
		roots = append(roots, filepath.Join(xdgData, AppID), filepath.Join(xdgConf, AppID))
	}

	// Older builds used a different identifier layout; check a couple of shapes.
	var extra []string
	for _, root := range roots {
		parent := filepath.Dir(root)
		extra = append(extra, filepath.Join(parent, "Buzz"), filepath.Join(parent, "buzz"))
	}

	var out []string
	for _, r := range append(roots, extra...) {
		out = append(out, filepath.Join(r, "agents", "managed-agents.json"))
	}
	return out
}

// FindStore resolves the registry path: an explicit override if given,
// otherwise whichever single candidate path exists on disk.
func FindStore(override string) (string, error) {
	if override != "" {
		path := expandHome(override)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("--store points at a file that does not exist: %s", path)
		}
		return path, nil
	}

	var found []string
	for _, p := range CandidateStores() {
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}

	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		var b strings.Builder
		b.WriteString("could not find managed-agents.json.\nLooked in:\n")
		for _, p := range CandidateStores() {
			b.WriteString("  " + p + "\n")
		}
		b.WriteString("Open Buzz Desktop once, then re-run. If it still fails, locate the\n")
		b.WriteString("file yourself and pass --store /path/to/managed-agents.json")
		return "", fmt.Errorf("%s", b.String())
	default:
		var b strings.Builder
		b.WriteString("found more than one registry:\n")
		for _, p := range found {
			b.WriteString("  " + p + "\n")
		}
		b.WriteString("pass the right one with --store")
		return "", fmt.Errorf("%s", b.String())
	}
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	return p
}

// LoadStore reads and parses the registry. Numbers decode as json.Number so
// a later WriteStore round-trips them byte-for-byte instead of drifting
// through float64.
func LoadStore(path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var rows []Row
	if err := dec.Decode(&rows); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON (%w). Refusing to touch it", path, err)
	}
	return rows, nil
}

// WriteStore writes the registry atomically: temp file in the same
// directory (so the rename is on one filesystem), mode 0600 throughout.
func WriteStore(path string, rows []Row) error {
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { os.Remove(tmpName) }

	if _, err := tmp.Write(append(payload, '\n')); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return os.Chmod(path, 0o600)
}

// BackupsFor lists this registry's backups, newest first.
func BackupsFor(path string) []string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	prefix := base + ".bak-"
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

// BackupPath builds a timestamped backup path next to the registry.
func BackupPath(path, stamp string) string {
	return path + ".bak-" + stamp
}

// CopyFile copies src to dst, preserving neither mode nor timestamps beyond
// what the destination's own create call sets (0600, matching the registry).
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

// NamesIn summarizes a registry's agent rows by display name, for
// before/after comparisons (status, rollback previews).
func NamesIn(rows []Row) []string {
	var names []string
	for _, r := range rows {
		if pubkey, _ := r["pubkey"].(string); pubkey == "" {
			continue
		}
		name, _ := r["name"].(string)
		if name == "" {
			name = "(unnamed)"
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return []string{"(no agents)"}
	}
	return names
}
