package bearing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

func (s *Store) Backup(ctx context.Context, path string) error {
	if path == "" {
		return fail("invalid_input", "Choose a new backup file.")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	ok := false
	defer func() {
		if !ok {
			os.Remove(path)
		}
	}()
	if _, err := s.DB.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *Store) Export(ctx context.Context, path string) error {
	if path == "" {
		return fail("invalid_input", "Choose a new export file.")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	ok := false
	defer func() {
		if !ok {
			os.Remove(path)
		}
	}()
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	out := map[string]any{"schema_version": 1, "bearing_version": Version, "repository": s.Root}
	for _, table := range []string{"tasks", "checkpoints", "documents", "memories"} {
		rows, err := tx.QueryContext(ctx, "SELECT data FROM "+table+" ORDER BY id")
		if err != nil {
			return err
		}
		records := []json.RawMessage{}
		for rows.Next() {
			var data string
			if err := rows.Scan(&data); err != nil {
				rows.Close()
				return err
			}
			records = append(records, json.RawMessage(data))
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		out[table] = records
	}
	if err := json.NewEncoder(file).Encode(out); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

// Restore accepts a physical backup only and never replaces an existing store.
func Restore(ctx context.Context, repo, source string) error {
	if source == "" {
		return fail("invalid_input", "Choose a Bearing SQLite backup.")
	}
	repository, err := ResolveRepository(ctx, repo)
	if err != nil {
		return err
	}
	destination, err := statePath(repository.Path)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fail("conflict", "Restore requires a checkout without an existing Bearing database.")
	}
	path, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	// Copy to a private file before validation so changes to the supplied file cannot alter the validated result.
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), "restore-*.db")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := io.Copy(temporary, input); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	u := url.URL{Scheme: "file", Path: temporary.Name()}
	values := url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)"}}
	u.RawQuery = values.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return err
	}
	defer db.Close()
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	// Older supported versions are migrated by Init after publication.
	if version < 1 || version > schemaVersion {
		return fail("invalid_input", "Backup uses an unsupported schema version.")
	}
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fail("invalid_input", "Backup failed SQLite integrity checks.")
	}
	var identity string
	if err := db.QueryRowContext(ctx, "SELECT id FROM store_meta WHERE version=?", version).Scan(&identity); err != nil {
		return fail("invalid_input", "This is not a Bearing backup.")
	}
	for _, table := range []string{"tasks", "checkpoints", "documents", "memories", "memory_anchors", "memory_search", "write_receipts"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			return fmt.Errorf("invalid backup table %s: %w", table, err)
		}
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	bad := rows.Next()
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if bad {
		return fail("invalid_input", "Backup contains broken record links.")
	}
	for _, table := range []string{"tasks", "checkpoints", "documents", "memories"} {
		var invalid int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table+" WHERE NOT json_valid(data)").Scan(&invalid); err != nil {
			return err
		}
		if invalid != 0 {
			return fail("invalid_input", "Backup contains invalid record JSON.")
		}
	}
	checks := []string{
		`SELECT count(*) FROM write_receipts WHERE NOT json_valid(outcome)`,
		`SELECT count(*) FROM tasks t WHERE json_extract(data,'$.id') IS NOT id OR json_extract(data,'$.version') IS NOT version OR json_extract(data,'$.status') IS NOT status OR version < 1 OR (version=1 AND json_extract(data,'$.latest_checkpoint_id') IS NOT NULL) OR (SELECT count(*) FROM checkpoints c WHERE c.task_id=t.id) != version-1 OR (version>1 AND NOT EXISTS(SELECT 1 FROM checkpoints c WHERE c.id=json_extract(t.data,'$.latest_checkpoint_id') AND c.task_id=t.id AND c.sequence=t.version-1))`,
		`SELECT count(*) FROM checkpoints WHERE json_extract(data,'$.id') IS NOT id OR json_extract(data,'$.task_id') IS NOT task_id OR json_extract(data,'$.sequence') IS NOT sequence`,
		`SELECT count(*) FROM memories WHERE json_extract(data,'$.id') IS NOT id OR json_extract(data,'$.version') IS NOT version OR json_extract(data,'$.status') IS NOT status`,
		`SELECT count(*) FROM documents d WHERE json_extract(data,'$.id') IS NOT id OR json_extract(data,'$.task_id') IS NOT task_id OR json_extract(data,'$.checkpoint_id') IS NOT checkpoint_id OR json_extract(data,'$.path') IS NOT path OR NOT EXISTS(SELECT 1 FROM checkpoints c WHERE c.id=d.checkpoint_id AND c.task_id=d.task_id)`,
		`SELECT count(*) FROM tasks t,json_each(t.data,'$.input_documents') ref WHERE NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=ref.value)`,
		`SELECT count(*) FROM documents d,json_each(d.data,'$.input_documents') ref WHERE NOT EXISTS(SELECT 1 FROM documents parent WHERE parent.id=ref.value)`,
		`SELECT count(*) FROM checkpoints c,json_each(c.data,'$.document_ids') ref WHERE NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=ref.value AND d.checkpoint_id=c.id)`,
		`SELECT count(*) FROM checkpoints c,json_each(c.data,'$.memory_refs') ref WHERE NOT EXISTS(SELECT 1 FROM memories m WHERE m.id=ref.value)`,
	}
	for _, query := range checks {
		var invalid int
		if err := db.QueryRowContext(ctx, query).Scan(&invalid); err != nil {
			return err
		}
		if invalid != 0 {
			return fail("invalid_input", "Backup contains inconsistent records or missing relationships.")
		}
	}
	documents, err := db.QueryContext(ctx, "SELECT data FROM documents")
	if err != nil {
		return err
	}
	for documents.Next() {
		var data string
		var doc Document
		if err := documents.Scan(&data); err != nil {
			documents.Close()
			return err
		}
		if err := json.Unmarshal([]byte(data), &doc); err != nil {
			documents.Close()
			return err
		}
		// note: keep this two-use digest calculation inline with document capture.
		if doc.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(doc.Body))) {
			documents.Close()
			return fail("invalid_input", "Backup document content does not match its saved digest.")
		}
	}
	err = documents.Err()
	documents.Close()
	if err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	// A hard link publishes the complete validated file atomically and refuses replacement.
	if err := os.Link(temporary.Name(), destination); err != nil {
		return err
	}
	store, err := Init(ctx, repository.Path)
	if err != nil {
		return fmt.Errorf("backup restored, but initialization needs attention: %w", err)
	}
	return store.Close()
}
