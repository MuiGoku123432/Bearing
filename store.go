package bearing

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB            *sql.DB
	Root          string
	SQLiteVersion string
}

const schema = `
CREATE TABLE store_meta (id TEXT PRIMARY KEY, version INTEGER NOT NULL);
CREATE TABLE tasks (id TEXT PRIMARY KEY, status TEXT NOT NULL, version INTEGER NOT NULL, data TEXT NOT NULL);
CREATE TABLE checkpoints (id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id), sequence INTEGER NOT NULL, data TEXT NOT NULL, UNIQUE(task_id,sequence));
CREATE TABLE documents (id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id), checkpoint_id TEXT NOT NULL REFERENCES checkpoints(id), path TEXT NOT NULL, data TEXT NOT NULL);
CREATE INDEX documents_task ON documents(task_id);
CREATE TABLE memories (id TEXT PRIMARY KEY, status TEXT NOT NULL, version INTEGER NOT NULL, data TEXT NOT NULL);
CREATE TABLE memory_anchors (memory_id TEXT NOT NULL REFERENCES memories(id), kind TEXT NOT NULL, value TEXT NOT NULL, PRIMARY KEY(memory_id,kind,value));
CREATE INDEX anchors_value ON memory_anchors(kind,value);
CREATE VIRTUAL TABLE memory_search USING fts5(id UNINDEXED, claim, evidence);
CREATE TABLE write_receipts (request_id TEXT PRIMARY KEY, operation TEXT NOT NULL, digest TEXT NOT NULL, outcome TEXT NOT NULL);
PRAGMA user_version=1;
`

func Init(ctx context.Context, repo string) (*Store, error) {
	r, err := ResolveRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	path, err := statePath(r.Path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	// Ignore state before creating it so repository observations exclude the database.
	// The state directory ignores itself, so initialization leaves tracked files unchanged.
	dir, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	if info, err := dir.Lstat(".gitignore"); err == nil && !info.Mode().IsRegular() {
		return nil, fail("invalid_input", "Refusing to replace .bearing/.gitignore because it is not a regular file.")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err = dir.WriteFile(".gitignore", []byte("*\n"), 0600); err != nil {
		return nil, err
	}
	return openStore(ctx, r.Path, true)
}

func Open(ctx context.Context, repo string) (*Store, error) {
	r, err := ResolveRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	return openStore(ctx, r.Path, false)
}

func openStore(ctx context.Context, root string, initialize bool) (*Store, error) {
	path, err := statePath(root)
	if err != nil {
		return nil, err
	}
	if !initialize {
		if _, err = os.Stat(path); os.IsNotExist(err) {
			return nil, fail("not_initialized", "Run bearing init in this checkout first.")
		} else if err != nil {
			return nil, err
		}
	}
	mode := "rw"
	if initialize {
		mode = "rwc"
	}
	// Connection pragmas are applied whenever database/sql recreates a connection.
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {mode}, "_txlock": {"immediate"}, "_pragma": {"foreign_keys(1)", "busy_timeout(2000)", "synchronous(FULL)"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{DB: db, Root: root}
	ok := false
	defer func() {
		if !ok {
			db.Close()
		}
	}()
	if err = db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&s.SQLiteVersion); err != nil {
		return nil, err
	}
	var major, minor, patch int
	if _, err = fmt.Sscanf(s.SQLiteVersion, "%d.%d.%d", &major, &minor, &patch); err != nil || major < 3 || (major == 3 && (minor < 51 || minor == 51 && patch < 3)) {
		return nil, fail("unsupported_schema", "Bearing needs SQLite 3.51.3 or later for its WAL durability requirements.")
	}
	var version int
	if err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if version > 1 || version < 0 {
		return nil, fail("unsupported_schema", "This database needs a different Bearing version. It was not migrated.")
	}
	if version == 0 && !initialize {
		return nil, fail("not_initialized", "Database initialization is incomplete. Run bearing init.")
	}
	if version == 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if err = tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			return nil, err
		}
		if version == 0 {
			var tables int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
				return nil, err
			}
			if tables != 0 {
				return nil, fail("unsupported_schema", "Refusing to initialize an unrecognized database.")
			}
			if _, err = tx.ExecContext(ctx, schema); err != nil {
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO store_meta(id,version) VALUES(?,1)", NewID()); err != nil {
				return nil, err
			}
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	}
	var journal string
	if err = db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journal); err != nil {
		return nil, err
	}
	if journal != "wal" {
		return nil, fail("io_error", "SQLite could not enable WAL for this local database.")
	}
	if err = os.Chmod(path, 0600); err != nil {
		return nil, err
	}
	ok = true
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}
