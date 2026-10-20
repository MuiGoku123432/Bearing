package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	bearing "github.com/MuiGoku123432/Bearing"
)

func installTestRepo(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "checkout with spaces")
	if out, err := exec.Command("git", "init", "-q", path).CombinedOutput(); err != nil {
		t.Fatalf("git: %s %v", out, err)
	}
	return path
}

func TestInstallPreservesSettingsAndRepeats(t *testing.T) {
	for _, test := range []struct{ host, major, config, instructions, skill, original string }{
		{"codex", "", ".codex/config.toml", "AGENTS.md", ".agents/skills/bearing/SKILL.md", "# Keep this comment\nmodel = \"example\"\n[mcp_servers.gortex]\ncommand = \"gortex\"\n"},
		{"claude", "", ".mcp.json", "CLAUDE.md", ".claude/skills/bearing/SKILL.md", `{"other":{"number":9007199254740993},"mcpServers":{"gortex":{"command":"gortex"},"bearing":{"type":"stdio","command":"/previous/bearing","args":[],"env":{"KEEP":"custom"},"timeout":12000}}}`},
		{"opencode", "1", "opencode.json", "AGENTS.md", ".opencode/skills/bearing/SKILL.md", `{"model":"example","mcp":{"gortex":{"type":"local","command":["gortex"]}}}`},
		{"opencode", "2", "opencode.json", "AGENTS.md", ".opencode/skills/bearing/SKILL.md", `{"model":"example","mcp":{"servers":{"gortex":{"type":"local","command":["gortex"]}}}}`},
	} {
		t.Run(test.host+test.major, func(t *testing.T) {
			ctx := context.Background()
			root := installTestRepo(t)
			config := filepath.Join(root, test.config)
			if err := os.MkdirAll(filepath.Dir(config), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(config, []byte(test.original), 0640); err != nil {
				t.Fatal(err)
			}
			instructions := []byte("# Team rules\nKeep these instructions.\n")
			if err := os.WriteFile(filepath.Join(root, test.instructions), instructions, 0644); err != nil {
				t.Fatal(err)
			}
			var output, diagnostics bytes.Buffer
			args := []string{"install", "--repo", root, "--host", test.host, "--dry-run"}
			if test.major != "" {
				args = append(args, "--opencode-version", test.major)
			}
			if err := run(ctx, args, strings.NewReader(""), &output, &diagnostics); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "Preview only") {
				t.Fatal(output.String())
			}
			if _, err := os.Stat(filepath.Join(root, ".bearing")); !os.IsNotExist(err) {
				t.Fatal("preview initialized state")
			}
			before, _ := os.ReadFile(config)
			if string(before) != test.original {
				t.Fatal("preview changed settings")
			}
			report, err := installSetup(ctx, root, test.host, test.major, false)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(report, "Recovery files:") {
				t.Fatal(report)
			}
			store, err := bearing.Open(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			store.Close()
			data, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			if test.host == "codex" {
				if !bytes.HasPrefix(data, before) {
					t.Fatal("existing TOML changed")
				}
			} else {
				var previous, current map[string]json.RawMessage
				if err := json.Unmarshal(before, &previous); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &current); err != nil {
					t.Fatal(err)
				}
				key := "mcp"
				if test.host == "claude" {
					key = "mcpServers"
				}
				for name, value := range previous {
					if name == key {
						continue
					}
					var a, b bytes.Buffer
					if err := json.Compact(&a, value); err != nil {
						t.Fatal(err)
					}
					if err := json.Compact(&b, current[name]); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(a.Bytes(), b.Bytes()) {
						t.Fatalf("setting %s changed", name)
					}
				}
				if !bytes.Contains(data, []byte(`"gortex"`)) {
					t.Fatal("Gortex setting lost")
				}
			}
			if test.host == "claude" && (!bytes.Contains(data, []byte(`"KEEP"`)) || !bytes.Contains(data, []byte(`12000`))) {
				t.Fatal("Bearing custom settings lost")
			}
			if !bytes.Contains(data, []byte(root)) {
				t.Fatal("missing canonical checkout binding")
			}
			info, _ := os.Stat(config)
			if info.Mode().Perm() != 0640 {
				t.Fatal("permissions changed")
			}
			entry, _ := os.ReadFile(filepath.Join(root, test.instructions))
			if !bytes.HasPrefix(entry, instructions) {
				t.Fatal("team instructions lost")
			}
			skill, _ := os.ReadFile(filepath.Join(root, test.skill))
			if !bytes.Contains(skill, []byte(bearing.Rules().Text)) || !bytes.Contains(skill, []byte(bearing.Rules().Workflow)) {
				t.Fatal("skill lost embedded policy or workflow")
			}
			backups, _ := filepath.Glob(filepath.Join(root, ".bearing", "install-backups", "*", test.config))
			if len(backups) != 1 {
				t.Fatalf("backups: %v", backups)
			}
			backup, _ := os.ReadFile(backups[0])
			if !bytes.Equal(backup, before) {
				t.Fatal("backup is not original bytes")
			}
			report, err = installSetup(ctx, root, test.host, test.major, false)
			if err != nil || !strings.Contains(report, "Already installed") {
				t.Fatalf("repeat: %s %v", report, err)
			}
			again, _ := os.ReadFile(config)
			infoAgain, _ := os.Stat(config)
			if !bytes.Equal(again, data) || !infoAgain.ModTime().Equal(info.ModTime()) {
				t.Fatal("repeat rewrote config")
			}
		})
	}
}

func TestInstallPreflightRefusesAmbiguousFiles(t *testing.T) {
	for _, test := range []struct{ name, host, major, path, body string }{
		{"invalid JSON", "claude", "", ".mcp.json", `{oops`},
		{"duplicate JSON", "claude", "", ".mcp.json", `{"mcpServers":{},"mcpServers":{"other":{}}}`},
		{"null JSON", "claude", "", ".mcp.json", `null`},
		{"null servers", "claude", "", ".mcp.json", `{"mcpServers":null}`},
		{"remote Bearing", "claude", "", ".mcp.json", `{"mcpServers":{"bearing":{"type":"http","url":"https://example.invalid"}}}`},
		{"null Bearing", "claude", "", ".mcp.json", `{"mcpServers":{"bearing":null}}`},
		{"V1 other server", "opencode", "2", "opencode.json", `{"mcp":{"gortex":{"type":"local","command":["gortex"]}}}`},
		{"JSONC", "opencode", "1", "opencode.jsonc", `{// comment\n}`},
		{"OpenCode migration", "opencode", "1", "opencode.json", `{"mcp":{"servers":{}}}`},
		{"unmanaged TOML", "codex", "", ".codex/config.toml", "[mcp_servers.bearing]\ncommand = \"other\"\n"},
		{"inline TOML", "codex", "", ".codex/config.toml", "mcp_servers = {}\n"},
		{"multiline TOML", "codex", "", ".codex/config.toml", "instructions = '''text'''\n"},
		{"broken markers", "claude", "", "CLAUDE.md", "before\n<!-- bearing:start -->\nno end\n"},
		{"custom skill", "claude", "", ".claude/skills/bearing/SKILL.md", "my custom skill\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := installTestRepo(t)
			path := filepath.Join(root, test.path)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := installSetup(context.Background(), root, test.host, test.major, false); err == nil {
				t.Fatal("ambiguous install succeeded")
			}
			if _, err := os.Stat(filepath.Join(root, ".bearing")); !os.IsNotExist(err) {
				t.Fatal("preflight failure changed state")
			}
			data, _ := os.ReadFile(path)
			if string(data) != test.body {
				t.Fatal("preflight failure changed input")
			}
		})
	}
}

func TestInstallRollsBackWhenSkillWriteFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("directory permissions do not constrain root")
	}
	root := installTestRepo(t)
	original := map[string]string{
		".mcp.json": `{"mcpServers":{"gortex":{"command":"gortex"}}}`,
		"CLAUDE.md": "Existing team rules.\n",
	}
	for path, body := range original {
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0640); err != nil {
			t.Fatal(err)
		}
	}
	skillDir := filepath.Join(root, ".claude", "skills", "bearing")
	if err := os.MkdirAll(skillDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(skillDir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(skillDir, 0700)
	_, err := installSetup(context.Background(), root, "claude", "", false)
	if err == nil || !strings.Contains(err.Error(), "recovery files:") {
		t.Fatalf("expected write failure: %v", err)
	}
	for path, body := range original {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || string(data) != body {
			t.Fatalf("rollback lost %s: %v", path, err)
		}
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil || info.Mode().Perm() != 0640 {
			t.Fatalf("rollback changed mode: %v", err)
		}
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("partial skill remained")
	}
	if _, err := os.Stat(filepath.Join(root, ".bearing", "install.lock")); !os.IsNotExist(err) {
		t.Fatal("failed install retained its lock")
	}
	if err := os.Chmod(skillDir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := installSetup(context.Background(), root, "claude", "", false); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

func TestInstallRefusesSymlinksAndSharesInstructions(t *testing.T) {
	ctx := context.Background()
	root := installTestRepo(t)
	outside := filepath.Join(t.TempDir(), "settings")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".mcp.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := installSetup(ctx, root, "claude", "", false); err == nil {
		t.Fatal("followed symlink")
	}
	if err := os.Remove(filepath.Join(root, ".mcp.json")); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"codex", "claude", "opencode", "codex"} {
		major := ""
		if host == "opencode" {
			major = "1"
		}
		if _, err := installSetup(ctx, root, host, major, false); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if bytes.Count(data, []byte("<!-- bearing:start -->")) != 1 {
		t.Fatal("clients duplicated shared instructions")
	}
	original, _ := os.ReadFile(outside)
	if string(original) != "{}" {
		t.Fatal("changed symlink target")
	}
}
