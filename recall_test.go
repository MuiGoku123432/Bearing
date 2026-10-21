package bearing

import (
	"context"
	"testing"
)

// Known-answer recall fixtures: each query names the memory that should appear in
// the top three results. A miss reason marks a known failure of lexical recall; those
// cases skip, so the suite passes and the misses stay visible. A known miss that
// starts to hit fails, so the reason is removed.
func TestRecallKnownAnswers(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	memories := []struct{ name, claim string }{
		{"retry", "Retrying a checkpoint with the same request ID replays the saved receipt"},
		{"sqlite", "Use SQLite transactions so the checkpoint and its receipt commit together"},
		{"flaky", "The intermittent CI failure in restore comes from a slow disk"},
		{"login", "The login handler must reject expired session tokens"},
		{"database", "Keep one database connection per process"},
		{"pascal", "ParseConfig must reject unknown keys"},
		{"camel", "Call loadSettings once at startup"},
		{"snake", "write_receipt stores the outcome JSON"},
		{"kebab", "The feature flag new-checkout stays off in production"},
		{"dotted", "store.go owns schema migrations"},
		{"qualified", "bearing.Rules returns the embedded policy"},
		{"branch", "Never commit directly to the main branch"},
		{"dependencies", "Do not add new dependencies without approval"},
		{"format", "Format Go code with gofmt, not a custom formatter"},
	}
	ids := map[string]string{}
	for _, m := range memories {
		out, err := s.Remember(ctx, RememberInput{Kind: "lesson", Claim: m.claim, Basis: "observed", Source: "Recall fixture"})
		if err != nil {
			t.Fatal(err)
		}
		ids[m.name] = out.ID
	}
	cases := []struct{ form, query, want, miss string }{
		{"lexical", "replay a request", "retry", ""},
		{"lexical", "commit to main", "branch", ""},
		{"stemmed", "retries", "retry", ""},
		{"paraphrase", "flaky test", "flaky", "no word shared with \"intermittent CI failure\"; lexical recall has no semantics"},
		{"paraphrase", "how many connections should a process open", "database", ""},
		{"paraphrase", "atomic progress writes", "sqlite", "no word shared with \"SQLite transactions ... commit together\""},
		{"synonym", "DB", "database", "no synonym table: DB is not database"},
		{"synonym", "auth", "login", "no synonym table: auth is not login"},
		{"synonym", "third-party library", "dependencies", "no synonym table: library is not dependency"},
		{"PascalCase", "ParseConfig", "pascal", ""},
		{"PascalCase", "parse config", "pascal", "unicode61 keeps ParseConfig as one token"},
		{"camelCase", "loadSettings", "camel", ""},
		{"camelCase", "load settings", "camel", "unicode61 keeps loadSettings as one token"},
		{"camelCase", "parseConfig", "pascal", ""},
		{"snake_case", "write_receipt", "snake", ""},
		{"snake_case", "parse_config", "pascal", "the query splits into parse and config; the stored ParseConfig is one token"},
		{"snake_case", "writeReceipt", "snake", "the query is one token; the stored write_receipt splits into write and receipt"},
		{"kebab-case", "new-checkout", "kebab", ""},
		{"dotted", "store.go", "dotted", ""},
		{"dotted", "bearing.Rules", "qualified", ""},
	}
	hits := 0
	for _, c := range cases {
		result, err := s.Recall(ctx, RecallInput{Query: c.query, Limit: 3})
		if err != nil {
			t.Fatal(err)
		}
		hit := false
		for _, memory := range result.Memories {
			hit = hit || memory.ID == ids[c.want]
		}
		if hit {
			hits++
		}
		t.Run(c.form+"/"+c.query, func(t *testing.T) {
			switch {
			case hit && c.miss != "":
				t.Fatalf("known miss now finds %s; remove its miss reason", c.want)
			case !hit && c.miss == "":
				t.Fatalf("%q did not return %s in the top three", c.query, c.want)
			case !hit:
				t.Skip("known miss: " + c.miss)
			}
		})
	}
	t.Logf("recall hit@3: %d/%d", hits, len(cases))
}
