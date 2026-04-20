package cmd_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/6xiaowu9/asm/cmd"
)

// testEnv sets up an isolated ASM home with a config pointing claude at claudeHome.
func testEnv(t *testing.T) (asmHome, claudeHome string) {
	t.Helper()
	asmHome = t.TempDir()
	claudeHome = t.TempDir()

	cfgContent := fmt.Sprintf(`asm_home = %q
link_mode = "auto"
git_cache_dir = %q
default_agents = ["claude"]

[agent_paths]
claude = %q
`, asmHome, filepath.Join(asmHome, "cache", "git"), claudeHome)

	cfgPath := filepath.Join(asmHome, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASM_HOME", asmHome)
	return asmHome, claudeHome
}

// run executes the CLI with the given args, returning combined output and any error.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := cmd.ExecuteWithWriter(&buf, args...)
	return buf.String(), err
}

func TestSkillsInstallAndList(t *testing.T) {
	asmHome, _ := testEnv(t)
	_ = asmHome

	srcDir := filepath.Join(t.TempDir(), "my-skill")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "skill.md"), []byte("# my-skill"), 0o644)

	out, err := run(t, "skills", "install", srcDir)
	if err != nil {
		t.Fatalf("install: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "installed my-skill") {
		t.Errorf("install output = %q, want 'installed my-skill'", out)
	}

	out, err = run(t, "skills", "list")
	if err != nil {
		t.Fatalf("list: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "my-skill") {
		t.Errorf("list output = %q, want 'my-skill'", out)
	}
}

func TestSkillsEnableAndStatus(t *testing.T) {
	testEnv(t)

	srcDir := filepath.Join(t.TempDir(), "status-skill")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "skill.md"), []byte("# status-skill"), 0o644)

	_, err := run(t, "skills", "install", srcDir, "--agents", "claude")
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	out, err := run(t, "skills", "status", "--agent", "claude")
	if err != nil {
		t.Fatalf("status: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "status-skill") {
		t.Errorf("status output = %q, want 'status-skill'", out)
	}
}

func TestSkillsRemove(t *testing.T) {
	testEnv(t)

	srcDir := filepath.Join(t.TempDir(), "rm-skill")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "skill.md"), []byte("# rm-skill"), 0o644)

	_, err := run(t, "skills", "install", srcDir, "--agents", "claude")
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	out, err := run(t, "skills", "remove", "rm-skill")
	if err != nil {
		t.Fatalf("remove: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "removed rm-skill") {
		t.Errorf("remove output = %q", out)
	}

	out, err = run(t, "skills", "list")
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if strings.Contains(out, "rm-skill") {
		t.Errorf("rm-skill still listed after remove: %s", out)
	}
}

func TestSkillsDoctor_Healthy(t *testing.T) {
	testEnv(t)

	srcDir := filepath.Join(t.TempDir(), "doc-skill")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "skill.md"), []byte("# doc-skill"), 0o644)

	_, err := run(t, "skills", "install", srcDir, "--agents", "claude")
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	out, err := run(t, "skills", "doctor")
	if err != nil {
		t.Fatalf("doctor: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "healthy") {
		t.Errorf("doctor output = %q, want 'healthy'", out)
	}
}

func TestMCPInstallAndList(t *testing.T) {
	testEnv(t)

	srcDir := filepath.Join(t.TempDir(), "my-mcp")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "mcp.json"), []byte(`{}`), 0o644)

	out, err := run(t, "mcp", "install", srcDir)
	if err != nil {
		t.Fatalf("mcp install: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "installed my-mcp") {
		t.Errorf("mcp install output = %q, want 'installed my-mcp'", out)
	}

	out, err = run(t, "mcp", "list")
	if err != nil {
		t.Fatalf("mcp list: %v", err)
	}
	if !strings.Contains(out, "my-mcp") {
		t.Errorf("mcp list output = %q, want 'my-mcp'", out)
	}
}
