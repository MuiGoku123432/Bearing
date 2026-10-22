package bearing

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime/debug"
	"testing"
)

func TestVersionFallsBackToLinkerValue(t *testing.T) {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		t.Skip("this test binary carries module version " + info.Main.Version)
	}
	if got := Version(); got != "0.1.0-dev" {
		t.Fatalf("development fallback: %s", got)
	}
	version = "v9.9.9-test"
	defer func() { version = "" }()
	if got := Version(); got != "v9.9.9-test" {
		t.Fatalf("-ldflags value ignored: %s", got)
	}
}

func TestRulesCarryPlanningStandards(t *testing.T) {
	data, err := os.ReadFile("PLANNING.md")
	if err != nil {
		t.Fatal(err)
	}
	rules := Rules()
	if rules.Planning != string(data) || rules.PlanningID != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("rules must return the embedded planning standards and their SHA-256")
	}
}
