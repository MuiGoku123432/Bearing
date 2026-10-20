package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	bearing "github.com/MuiGoku123432/Bearing"
)

func installSetup(ctx context.Context, repo, host, major string, dryRun bool) (string, error) {
	if host != "codex" && host != "claude" && host != "opencode" {
		return "", fmt.Errorf("choose --host codex, claude, or opencode")
	}
	if host != "opencode" && major != "" {
		return "", fmt.Errorf("--opencode-version only applies to --host opencode")
	}
	if host == "opencode" {
		if major == "" {
			versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			data, err := exec.CommandContext(versionCtx, "opencode", "--version").Output()
			cancel()
			if err != nil {
				return "", fmt.Errorf("cannot detect OpenCode version; supply --opencode-version 1 or 2")
			}
			major = strings.SplitN(strings.TrimPrefix(strings.TrimSpace(string(data)), "v"), ".", 2)[0]
		}
		if major != "1" && major != "2" {
			return "", fmt.Errorf("supported OpenCode major versions are 1 and 2")
		}
	}
	repository, err := bearing.ResolveRepository(ctx, repo)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(repository.Path)
	if err != nil {
		return "", err
	}
	defer root.Close()
	// Reuse the reviewed setup output instead of maintaining a second set of
	// host settings, engineering rules or agent instructions.
	temporary, err := os.MkdirTemp("", "bearing-setup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temporary)
	bundle := filepath.Join(temporary, "bundle")
	if _, err := renderSetup(ctx, repository.Path, host, bundle); err != nil {
		return "", err
	}
	config, fragment, instructions, entry, skill := ".codex/config.toml", "config.fragment.toml", "AGENTS.md", "AGENTS.fragment.md", ".agents/skills/bearing/SKILL.md"
	if host == "claude" {
		config, fragment, instructions, entry, skill = ".mcp.json", "mcp.fragment.json", "CLAUDE.md", "CLAUDE.fragment.md", ".claude/skills/bearing/SKILL.md"
	}
	if host == "opencode" {
		config, fragment, skill = "opencode.json", "opencode.fragment.json", ".opencode/skills/bearing/SKILL.md"
		if major == "2" {
			fragment = "opencode-v2.fragment.json"
		}
		if _, err := root.Lstat("opencode.jsonc"); !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("opencode.jsonc exists or cannot be inspected; use bearing setup to merge it manually")
		}
	}
	type fileChange struct {
		path          string
		before, after []byte
		mode          fs.FileMode
	}
	var changes []fileChange
	for i, path := range []string{config, instructions, skill} {
		before, mode, err := readInstallFile(root, path)
		if err != nil {
			return "", err
		}
		name := []string{fragment, entry, "SKILL.md"}[i]
		after, err := os.ReadFile(filepath.Join(bundle, name))
		if err != nil {
			return "", err
		}
		switch i {
		case 0:
			after, err = installConfig(before, after, host, major)
		case 1:
			var merged string
			merged, err = installBlock(string(before), string(after), "<!-- bearing:start -->", "<!-- bearing:end -->")
			after = []byte(merged)
		case 2:
			const marker = "<!-- bearing:installed-skill -->\n"
			// Preserve frontmatter at the top of a discoverable skill.
			after = append(after, []byte("\n"+marker)...)
			if before != nil && !bytes.Contains(before, []byte(marker)) && !bytes.Equal(before, bytes.TrimSuffix(after, []byte("\n"+marker))) {
				err = fmt.Errorf("%s already contains a custom skill; move it or use bearing setup", path)
			}
		}
		if err != nil {
			return "", fmt.Errorf("%s: %w", path, err)
		}
		if !bytes.Equal(before, after) {
			changes = append(changes, fileChange{path, before, after, mode})
		}
	}
	var report strings.Builder
	fmt.Fprintf(&report, "Bearing for %s in %s\n", host, repository.Path)
	for _, change := range changes {
		action := "Create"
		if change.before != nil {
			action = "Update (back up first)"
		}
		fmt.Fprintf(&report, "%s: %s\n", action, change.path)
	}
	if dryRun {
		report.WriteString("Preview only. Apply without --dry-run to initialize Bearing and write these files.\n")
	} else {
		store, err := bearing.Init(ctx, repository.Path)
		if err != nil {
			return "", err
		}
		if err := store.Close(); err != nil {
			return "", err
		}
		lock, err := root.OpenFile(".bearing/install.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return "", fmt.Errorf("cannot acquire .bearing/install.lock; another install may be running: %w", err)
		}
		lock.Close()
		defer root.Remove(".bearing/install.lock")
		backup := filepath.Join(".bearing", "install-backups", bearing.NewID())
		var created, updated []string
		// Finish every backup before changing any client file.
		for _, change := range changes {
			current, _, err := readInstallFile(root, change.path)
			if err != nil || !bytes.Equal(current, change.before) || (current == nil) != (change.before == nil) {
				return "", fmt.Errorf("%s changed during installation; retry after reviewing it", change.path)
			}
			if change.before == nil {
				created = append(created, change.path)
				continue
			}
			updated = append(updated, change.path)
			if err := writeInstallFile(root, filepath.Join(backup, change.path), change.before, 0600); err != nil {
				return "", err
			}
		}
		if len(changes) > 0 {
			manifest, _ := json.MarshalIndent(map[string]any{"host": host, "repository": repository.Path, "created": created, "updated": updated}, "", "  ")
			if err := writeInstallFile(root, filepath.Join(backup, "INSTALL.json"), append(manifest, '\n'), 0600); err != nil {
				return "", err
			}
		}
		for i, change := range changes {
			err := ctx.Err()
			if err == nil {
				// note: repeat the pre-backup check immediately before replacement.
				current, _, readErr := readInstallFile(root, change.path)
				if readErr != nil || !bytes.Equal(current, change.before) || (current == nil) != (change.before == nil) {
					err = fmt.Errorf("%s changed during installation; review it before retrying", change.path)
				}
			}
			if err == nil {
				err = writeInstallFile(root, change.path, change.after, change.mode)
			}
			if err == nil {
				continue
			}
			// Roll back completed replacements on ordinary errors. A process kill
			// may leave a partial install; the manifest and backups survive it.
			for j := i - 1; j >= 0; j-- {
				previous := changes[j]
				current, _, readErr := readInstallFile(root, previous.path)
				if readErr != nil || !bytes.Equal(current, previous.after) {
					err = errors.Join(err, fmt.Errorf("%s changed again; kept it for manual recovery", previous.path))
					continue
				}
				if previous.before == nil {
					err = errors.Join(err, root.Remove(previous.path))
				} else {
					err = errors.Join(err, writeInstallFile(root, previous.path, previous.before, previous.mode))
				}
			}
			return "", fmt.Errorf("installation failed; recovery files: %s: %w", filepath.Join(repository.Path, backup), err)
		}
		if len(changes) == 0 {
			report.WriteString("Already installed. No client files changed.\n")
		} else {
			fmt.Fprintf(&report, "Installed. Recovery files: %s\n", filepath.Join(repository.Path, backup))
		}
	}
	report.WriteString("Keep this executable at its current path. Restart the client and approve project MCP settings if prompted.\n")
	if _, err := exec.LookPath("gortex"); err != nil {
		report.WriteString("Gortex is missing from PATH. Install and connect native Gortex before source work.\n")
	} else {
		report.WriteString("Gortex executable found. Verify its native MCP connection and that it covers this checkout before source work.\n")
	}
	report.WriteString("Bearing does not register repositories with Gortex or change global client settings.")
	return report.String(), nil
}
