package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	contracts "github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/internal/policy"
)

func TestInitCmd(t *testing.T) {
	// Build the binary once
	cmd := exec.Command("go", "build", "-o", "osmcp_test_bin", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	defer os.Remove("osmcp_test_bin")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	binPath := filepath.Join(cwd, "osmcp_test_bin")

	t.Run("scaffolds dev-agent by default", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command(binPath, "--init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}

		policyPath := filepath.Join(tempDir, ".osmcp", "policy.toml")
		content, err := os.ReadFile(policyPath)
		if err != nil {
			t.Fatalf("failed to read policy file: %v", err)
		}

		contentStr := string(content)
		if !contains(contentStr, "allow_mutation = true") {
			t.Errorf("expected dev-agent profile (allow_mutation = true), got something else")
		}
	})

	t.Run("prevents overwrite", func(t *testing.T) {
		tempDir := t.TempDir()

		// Run first time
		cmd1 := exec.Command(binPath, "--init")
		cmd1.Dir = tempDir
		if err := cmd1.Run(); err != nil {
			t.Fatalf("expected first run to succeed, got error: %v", err)
		}

		// Run second time, should fail
		cmd2 := exec.Command(binPath, "--init")
		cmd2.Dir = tempDir
		if err := cmd2.Run(); err == nil {
			t.Fatalf("expected second run to fail with overwrite error")
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command(binPath, "--init", "--profile", "fake-profile")
		cmd.Dir = tempDir
		if err := cmd.Run(); err == nil {
			t.Fatalf("expected fail with invalid profile")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s != "" && substr != "" && sContains(s, substr)
}

func sContains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPolicyTemplatesValid(t *testing.T) {
	templates := []string{"read-only", "dev-agent", "ci-agent", "review-agent"}
	cwd, _ := os.Getwd()
	repoRoot := filepath.Dir(filepath.Dir(cwd))

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			path := filepath.Join(repoRoot, "templates", "policies", tmpl+".toml")
			p, err := policy.LoadFromFile(path)
			if err != nil {
				t.Fatalf("failed to load template %s: %v", tmpl, err)
			}

			// We can't strictly validate missing allowed_root because user must fill it, but we can do a basic check
			if p.PolicyConfig.AllowedRoot == "" {
				t.Errorf("expected allowed_root to be defined in %s", tmpl)
			}
		})
	}
}

func TestExampleJSONDeserialization(t *testing.T) {
	cwd, _ := os.Getwd()
	repoRoot := filepath.Dir(filepath.Dir(cwd))
	examplesDir := filepath.Join(repoRoot, "templates", "examples")

	files, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("failed to read examples dir: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(examplesDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", file.Name(), err)
			}

			var env contracts.Envelope
			if err := json.Unmarshal(content, &env); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", file.Name(), err)
			}

			if env.Version != "1" {
				t.Errorf("expected version 1, got %s", env.Version)
			}

			if env.OK {
				if env.Error != nil {
					t.Errorf("success response should not have error object")
				}
			} else {
				if env.Error == nil {
					t.Errorf("error response should have error object")
				}
			}
		})
	}
}
