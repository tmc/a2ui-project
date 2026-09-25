package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveOutputConfigDefaultLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/a2ui\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := resolveOutputConfig(dir, "", "a2ui", "a2uibuild", "", "v09")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.A2UIDir != "a2ui" {
		t.Fatalf("A2UIDir = %q, want a2ui", cfg.A2UIDir)
	}
	if cfg.A2UIBuildDir != "a2uibuild" {
		t.Fatalf("A2UIBuildDir = %q, want a2uibuild", cfg.A2UIBuildDir)
	}
	if cfg.A2UIImport != "example.com/a2ui/a2ui" {
		t.Fatalf("A2UIImport = %q, want example.com/a2ui/a2ui", cfg.A2UIImport)
	}
	if cfg.VersionImport != "example.com/a2ui/a2ui/v09" {
		t.Fatalf("VersionImport = %q, want example.com/a2ui/a2ui/v09", cfg.VersionImport)
	}
}

func TestResolveOutputConfigRootLayout(t *testing.T) {
	dir := t.TempDir()

	cfg, err := resolveOutputConfig(dir, "example.com/root", ".", "a2uibuild", "", "v09")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.A2UIDir != "." {
		t.Fatalf("A2UIDir = %q, want .", cfg.A2UIDir)
	}
	if cfg.A2UIImport != "example.com/root" {
		t.Fatalf("A2UIImport = %q, want example.com/root", cfg.A2UIImport)
	}
	if cfg.VersionImport != "example.com/root/v09" {
		t.Fatalf("VersionImport = %q, want example.com/root/v09", cfg.VersionImport)
	}
}

func TestResolveOutputConfigExplicitA2UIImport(t *testing.T) {
	dir := t.TempDir()

	cfg, err := resolveOutputConfig(dir, "", "generated/a2ui", "generated/a2uibuild", "example.com/custom/a2ui", "v091")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.A2UIImport != "example.com/custom/a2ui" {
		t.Fatalf("A2UIImport = %q, want example.com/custom/a2ui", cfg.A2UIImport)
	}
	if cfg.VersionImport != "example.com/custom/a2ui/v091" {
		t.Fatalf("VersionImport = %q, want example.com/custom/a2ui/v091", cfg.VersionImport)
	}
}

func TestGenerateSDKRootLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/root\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, stale := range staleSDKDirs {
		if err := os.MkdirAll(filepath.Join(dir, stale), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := generateSDK(dir, "example.com/root", ".", "a2uibuild", "", "", ""); err != nil {
		t.Fatal(err)
	}
	for _, stale := range staleSDKDirs {
		if _, err := os.Stat(filepath.Join(dir, stale)); !os.IsNotExist(err) {
			t.Fatalf("stale directory %s not removed (err = %v)", stale, err)
		}
	}

	checks := []struct {
		path string
		want string
	}{
		{"a2ui.go", `"example.com/root/v09"`},
		{"a2ui.go", "//go:fix inline\ntype ServerMessage = v09.ServerMessage\n"},
		{"a2ui.go", "func And(values []v09.DynamicBoolean) v09.DynamicBoolean {\n\treturn v09.And(values)\n}"},
		{filepath.Join("a2uibuild", "zz_builders.go"), `import "example.com/root/v09"`},
		{filepath.Join("a2uischema", "manager.go"), `"example.com/root/v09"`},
	}
	for _, check := range checks {
		data, err := os.ReadFile(filepath.Join(dir, check.path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), check.want) {
			t.Fatalf("%s does not contain %q", check.path, check.want)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "a2uischema", "gen.go")); !os.IsNotExist(err) {
		t.Fatalf("a2uischema/gen.go copied into SDK output (err = %v)", err)
	}

	for _, path := range []string{
		filepath.Join(dir, "a2ui.go"),
		filepath.Join(dir, "a2uischema", "manager.go"),
		filepath.Join(dir, "a2uibuild", "surface.go"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "github.com/a2ui-project/a2ui/go") {
			t.Fatalf("%s contains upstream module import", path)
		}
	}
}
