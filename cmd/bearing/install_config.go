package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Only the marked region belongs to Bearing. Everything around it stays verbatim.
func installBlock(old, body, start, end string) (string, error) {
	block := start + "\n" + strings.TrimSpace(body) + "\n" + end
	if !strings.Contains(old, start) && !strings.Contains(old, end) {
		separator := ""
		if old != "" {
			separator = "\n\n"
		}
		return old + separator + block + "\n", nil
	}
	if strings.Count(old, start) != 1 || strings.Count(old, end) != 1 {
		return "", fmt.Errorf("duplicate or incomplete Bearing markers; repair them before installing")
	}
	a, b := strings.Index(old, start), strings.Index(old, end)
	if b < a || (a > 0 && old[a-1] != '\n') || (b > 0 && old[b-1] != '\n') ||
		(a+len(start) < len(old) && old[a+len(start)] != '\n') ||
		(b+len(end) < len(old) && old[b+len(end)] != '\n') {
		return "", fmt.Errorf("Bearing markers must be complete, ordered lines")
	}
	return old[:a] + block + old[b+len(end):], nil
}

func installConfig(old, fragment []byte, host, major string) ([]byte, error) {
	if host == "codex" {
		const start, end = "# bearing:start", "# bearing:end"
		outside, err := installBlock(string(old), "", start, end)
		if err != nil {
			return nil, err
		}
		outside = strings.Replace(outside, start+"\n\n"+end, "", 1)
		// Avoid guessing at inline tables, escaped names or multiline TOML. The
		// render-only setup command handles these through a manual merge.
		if strings.Contains(outside, "bearing") || strings.Contains(outside, `\u`) || strings.Contains(outside, `\U`) ||
			strings.Contains(outside, `"""`) || strings.Contains(outside, "'''") {
			return nil, fmt.Errorf("Codex config has an unmanaged Bearing entry or complex TOML; use bearing setup to merge it manually")
		}
		for _, line := range strings.Split(outside, "\n") {
			key := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
			if strings.HasPrefix(strings.Trim(key, "\"'"), "mcp_servers") {
				return nil, fmt.Errorf("Codex config uses an inline or dotted MCP assignment; use bearing setup to merge it manually")
			}
		}
		merged, err := installBlock(string(old), string(fragment), start, end)
		return []byte(merged), err
	}
	if len(bytes.TrimSpace(old)) == 0 {
		old = []byte("{}")
	}
	// Reject duplicate keys instead of silently dropping a user's settings.
	decoder := json.NewDecoder(bytes.NewReader(old))
	decoder.UseNumber()
	if err := checkInstallJSON(decoder); err != nil {
		return nil, fmt.Errorf("config must be a JSON object with unique keys: %w; use bearing setup for a manual merge", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("config contains trailing JSON data")
	}
	var existing, generated map[string]json.RawMessage
	if err := json.Unmarshal(old, &existing); err != nil || existing == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	if err := json.Unmarshal(fragment, &generated); err != nil {
		return nil, err
	}
	keys := []string{"mcpServers"}
	if host == "opencode" {
		keys = []string{"mcp"}
		if major == "2" {
			keys = append(keys, "servers")
		}
		var mcp map[string]json.RawMessage
		if raw, ok := existing["mcp"]; ok {
			if err := json.Unmarshal(raw, &mcp); err != nil || mcp == nil {
				return nil, fmt.Errorf("mcp must be an object")
			}
		}
		if major == "1" && mcp["servers"] != nil || major == "2" && mcp["bearing"] != nil {
			return nil, fmt.Errorf("OpenCode config uses a different MCP layout; migrate it before installing")
		}
		if major == "2" {
			for key, raw := range mcp {
				var entry map[string]json.RawMessage
				if key != "servers" && json.Unmarshal(raw, &entry) == nil && entry["type"] != nil {
					return nil, fmt.Errorf("OpenCode has V1 servers outside mcp.servers; migrate them before installing")
				}
			}
		}
	}
	// A two-level path is sufficient for the three documented client formats.
	parents := []map[string]json.RawMessage{existing}
	wanted := generated
	for _, key := range keys {
		child := map[string]json.RawMessage{}
		if raw, ok := parents[len(parents)-1][key]; ok {
			if err := json.Unmarshal(raw, &child); err != nil || child == nil {
				return nil, fmt.Errorf("%s must be an object", key)
			}
		}
		parents = append(parents, child)
		var next map[string]json.RawMessage
		if err := json.Unmarshal(wanted[key], &next); err != nil {
			return nil, err
		}
		wanted = next
	}
	current := parents[len(parents)-1]
	entry := map[string]json.RawMessage{}
	if raw, ok := current["bearing"]; ok {
		if err := json.Unmarshal(raw, &entry); err != nil || entry == nil {
			return nil, fmt.Errorf("the existing Bearing server must be an object")
		}
		var transport string
		if raw, ok := entry["type"]; ok {
			if err := json.Unmarshal(raw, &transport); err != nil {
				return nil, fmt.Errorf("invalid Bearing transport type")
			}
		}
		if entry["url"] != nil || transport != "" && transport != "local" && transport != "stdio" {
			return nil, fmt.Errorf("an existing remote Bearing server needs a manual merge with bearing setup")
		}
	}
	previousJSON, _ := json.Marshal(entry)
	var desired map[string]json.RawMessage
	if err := json.Unmarshal(wanted["bearing"], &desired); err != nil {
		return nil, err
	}
	// Replace only the launch settings Bearing owns. Retain user env/timeouts.
	for key, value := range desired {
		entry[key] = value
	}
	updatedJSON, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(previousJSON, updatedJSON) {
		return old, nil
	}
	current["bearing"] = updatedJSON
	for i := len(keys) - 1; i >= 0; i-- {
		data, err := json.Marshal(parents[i+1])
		if err != nil {
			return nil, err
		}
		parents[i][keys[i]] = data
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	return append(data, '\n'), err
}

func checkInstallJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	keys := map[string]bool{}
	for decoder.More() {
		if delim == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || keys[name] {
				return fmt.Errorf("duplicate or invalid object key")
			}
			keys[name] = true
		}
		if err := checkInstallJSON(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
