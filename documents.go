package bearing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// captureDocuments reads only explicitly selected Markdown files within this checkout.
func (s *Store) captureDocuments(inputs []DocumentInput) ([]Document, error) {
	if len(inputs) > 8 {
		return nil, fail("invalid_input", "Save at most eight documents per checkpoint.")
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var docs []Document
	seen := map[string]bool{}
	for _, in := range inputs {
		if err := inputSize(in, 3000); err != nil {
			return nil, err
		}
		in.Path = filepath.Clean(in.Path)
		if len(in.Path) > 512 || !filepath.IsLocal(in.Path) || strings.ToLower(filepath.Ext(in.Path)) != ".md" || seen[in.Path] {
			return nil, fail("invalid_input", "Document paths must be unique Markdown files inside the checkout.")
		}
		if (in.Kind != "research" && in.Kind != "spec" && in.Kind != "plan") || strings.TrimSpace(in.Summary) == "" {
			return nil, fail("invalid_input", "Each document needs a research, spec, or plan kind and a summary.")
		}
		seen[in.Path] = true
		file, err := root.Open(in.Path)
		if err != nil {
			return nil, err
		}
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			file.Close()
			return nil, fail("invalid_input", "Documents must be regular files.")
		}
		body, readErr := io.ReadAll(io.LimitReader(file, 256*1024+1))
		file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(body) > 256*1024 || !utf8.Valid(body) {
			return nil, fail("invalid_input", "Documents must be UTF-8 Markdown no larger than 256 KiB.")
		}
		docs = append(docs, Document{ID: NewID(), DocumentInput: in, Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(body)), Body: string(body)})
	}
	return docs, nil
}

func (s *Store) document(ctx context.Context, id string) (Document, error) {
	var data string
	var doc Document
	err := s.DB.QueryRowContext(ctx, "SELECT data FROM documents WHERE id = ?", id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return doc, fail("not_found", "Document not found.")
	}
	if err != nil {
		return doc, err
	}
	err = json.Unmarshal([]byte(data), &doc)
	return doc, err
}

func (s *Store) documentWarning(doc Document) string {
	docs, err := s.captureDocuments([]DocumentInput{doc.DocumentInput})
	if err != nil {
		return "Document " + doc.Path + " is unavailable; the saved version remains accessible by ID."
	}
	if docs[0].Digest != doc.Digest {
		return "Document " + doc.Path + " changed. Reconcile the saved requirements before continuing."
	}
	return ""
}
