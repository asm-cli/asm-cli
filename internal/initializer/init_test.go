package initializer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/6xiaowu9/asm/internal/agent"
	"github.com/6xiaowu9/asm/internal/initializer"
)

// testEnv creates an isolated home directory with the given agent subdirs pre-created.
func testEnv(t *testing.T, agents ...string) (asmHome, homeDir string) {
	t.Helper()
	homeDir = t.TempDir()
	asmHome = filepath.Join(homeDir, ".asm")
	for _, a := range agents {
		_ = os.MkdirAll(filepath.Join(homeDir, "."+a), 0o755)
	}
	return asmHome, homeDir
}

func defaultOpts(asmHome, homeDir string) initializer.Options {
	return initializer.Options{
		AsmHome: asmHome,
		HomeDir: homeDir,
	}
}

func TestRun_CreatesDirectoryTree(t *testing.T) {
	asmHome, homeDir := testEnv(t)
	_, err := initializer.New().Run(defaultOpts(asmHome, homeDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{
		filepath.Join("store", "skills"),
		filepath.Join("store", "mcps"),
		filepath.Join("cache", "git"),
	} {
		if _, err := os.Stat(filepath.Join(asmHome, sub)); err != nil {
			t.Errorf("expected dir %s to exist: %v", sub, err)
		}
	}
}

func TestRun_WritesConfigToml(t *testing.T) {
	asmHome, homeDir := testEnv(t)
	opts := defaultOpts(asmHome, homeDir)

	res, err := initializer.New().Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ConfigWrote {
		t.Error("expected ConfigWrote=true on first run")
	}
	if _, err := os.Stat(filepath.Join(asmHome, "config.toml")); err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}

	// Second run: already exists.
	res2, err := initializer.New().Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res2.ConfigWrote {
		t.Error("expected ConfigWrote=false on second run")
	}
}

func TestRun_WritesAsmMDToAsmHome(t *testing.T) {
	asmHome, homeDir := testEnv(t)
	if _, err := initializer.New().Run(defaultOpts(asmHome, homeDir)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(asmHome, "ASM.md"))
	if err != nil {
		t.Fatalf("ASM.md not created: %v", err)
	}
	if !strings.Contains(string(data), "Agent Skills Manager") {
		t.Error("ASM.md missing expected content")
	}
}

func TestRun_AutoDetect_SkipsAbsent(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude") // only claude exists
	res, err := initializer.New().Run(defaultOpts(asmHome, homeDir))
	if err != nil {
		t.Fatal(err)
	}
	skipped := 0
	for _, ar := range res.Agents {
		if ar.Skipped {
			skipped++
		}
	}
	// codex, cursor, gemini should be skipped; claude should not.
	if skipped != 3 {
		t.Errorf("expected 3 skipped agents, got %d: %+v", skipped, res.Agents)
	}
	for _, ar := range res.Agents {
		if ar.Agent == agent.Claude && ar.Skipped {
			t.Error("claude should not be skipped")
		}
	}
}

func TestRun_ExplicitAgents(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude", "codex")
	opts := defaultOpts(asmHome, homeDir)
	opts.Agents = []agent.Agent{agent.Codex}

	res, err := initializer.New().Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Agents) != 1 || res.Agents[0].Agent != agent.Codex {
		t.Errorf("expected only codex in result, got %+v", res.Agents)
	}
}

func TestRun_NoPerAgentASMMd_Claude(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude")
	if _, err := initializer.New().Run(defaultOpts(asmHome, homeDir)); err != nil {
		t.Fatal(err)
	}
	// Claude no longer gets a per-agent ASM.md; canonical file lives in asmHome.
	dst := filepath.Join(homeDir, ".claude", "ASM.md")
	if _, err := os.Stat(dst); err == nil {
		t.Error("~/.claude/ASM.md should NOT be created; canonical file is in ~/.asm/ASM.md")
	}
	// Canonical file must exist.
	if _, err := os.Stat(filepath.Join(asmHome, "ASM.md")); err != nil {
		t.Errorf("canonical ~/.asm/ASM.md missing: %v", err)
	}
}

func TestRun_WritesAgentASMMd_Cursor(t *testing.T) {
	asmHome, homeDir := testEnv(t, "cursor")
	if _, err := initializer.New().Run(defaultOpts(asmHome, homeDir)); err != nil {
		t.Fatal(err)
	}
	// Must be in rules/, not directly in .cursor/.
	rulesPath := filepath.Join(homeDir, ".cursor", "rules", "ASM.md")
	directPath := filepath.Join(homeDir, ".cursor", "ASM.md")
	if _, err := os.Stat(rulesPath); err != nil {
		t.Errorf("~/.cursor/rules/ASM.md not created: %v", err)
	}
	if _, err := os.Stat(directPath); err == nil {
		t.Error("~/.cursor/ASM.md should not be created (rules/ dir is the mechanism)")
	}
}

func TestRun_InjectsRefIntoCLAUDEmd(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude")
	claudeHome := filepath.Join(homeDir, ".claude")
	// Pre-create CLAUDE.md without the ASM reference.
	_ = os.WriteFile(filepath.Join(claudeHome, "CLAUDE.md"), []byte("# My config\n"), 0o644)

	res, err := initializer.New().Run(defaultOpts(asmHome, homeDir))
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(claudeHome, "CLAUDE.md"))
	// Injection uses the full path: @<asmHome>/ASM.md
	expectedRef := "@" + filepath.Join(asmHome, "ASM.md")
	if !strings.Contains(string(data), expectedRef) {
		t.Errorf("expected %q in CLAUDE.md, got:\n%s", expectedRef, data)
	}
	for _, ar := range res.Agents {
		if ar.Agent == agent.Claude && !ar.Injected {
			t.Error("AgentResult.Injected should be true")
		}
	}
}

func TestRun_Idempotent_NoDoubleInjection(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude")
	opts := defaultOpts(asmHome, homeDir)

	if _, err := initializer.New().Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := initializer.New().Run(opts); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(homeDir, ".claude", "CLAUDE.md"))
	ref := "@" + filepath.Join(asmHome, "ASM.md")
	count := strings.Count(string(data), ref)
	if count != 1 {
		t.Errorf("ref appears %d times in CLAUDE.md, want exactly 1\ncontent: %s", count, data)
	}
}

func TestRun_ForceOverwritesCursorASMMd(t *testing.T) {
	// cursor still writes rules/ASM.md; verify --force overwrites it.
	asmHome, homeDir := testEnv(t, "cursor")
	cursorRules := filepath.Join(homeDir, ".cursor", "rules")
	_ = os.MkdirAll(cursorRules, 0o755)
	stub := "old content"
	_ = os.WriteFile(filepath.Join(cursorRules, "ASM.md"), []byte(stub), 0o644)

	// Without --force: file unchanged.
	opts := defaultOpts(asmHome, homeDir)
	if _, err := initializer.New().Run(opts); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(cursorRules, "ASM.md"))
	if string(data) != stub {
		t.Error("expected cursor rules/ASM.md unchanged without --force")
	}

	// With --force: file overwritten.
	opts.Force = true
	if _, err := initializer.New().Run(opts); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(cursorRules, "ASM.md"))
	if string(data) == stub {
		t.Error("expected cursor rules/ASM.md overwritten with --force")
	}
	if !strings.Contains(string(data), "Agent Skills Manager") {
		t.Error("overwritten file missing expected content")
	}
}

func TestRun_DryRun_WritesNothing(t *testing.T) {
	asmHome, homeDir := testEnv(t, "claude")
	opts := defaultOpts(asmHome, homeDir)
	opts.DryRun = true

	res, err := initializer.New().Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Error("Result.DryRun should be true")
	}

	// asmHome directory tree should NOT exist.
	if _, err := os.Stat(filepath.Join(asmHome, "store")); err == nil {
		t.Error("store/ should not be created in dry-run mode")
	}
	// Per-agent ASM.md should NOT exist.
	if _, err := os.Stat(filepath.Join(homeDir, ".claude", "ASM.md")); err == nil {
		t.Error("~/.claude/ASM.md should not be created in dry-run mode")
	}
}

func TestRun_CursorNoInjection(t *testing.T) {
	asmHome, homeDir := testEnv(t, "cursor")
	if _, err := initializer.New().Run(defaultOpts(asmHome, homeDir)); err != nil {
		t.Fatal(err)
	}

	// None of the other config files should be created.
	for _, name := range []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md"} {
		if _, err := os.Stat(filepath.Join(homeDir, ".cursor", name)); err == nil {
			t.Errorf("~/.cursor/%s should not be created for cursor agent", name)
		}
	}
	// AgentResult.Injected must be false.
	res, _ := initializer.New().Run(defaultOpts(asmHome, homeDir))
	for _, ar := range res.Agents {
		if ar.Agent == agent.Cursor && ar.Injected {
			t.Error("cursor agent should never have Injected=true")
		}
	}
}
