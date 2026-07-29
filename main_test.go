// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaPath returns a path to one of the test schemas used by the codegen
// package tests, reusing them here to avoid duplication.
func schemaPath(name string) string {
	return filepath.Join("test", "schemas", name)
}

func TestAllThreeFilesGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, run(schemaPath("chatapp.yaml"), tmpDir, "messages"))

	for _, rel := range []string{
		filepath.Join("go", "messages.go"),
		filepath.Join("c", "messages.h"),
		filepath.Join("c", "messages.c"),
	} {
		fi, err := os.Stat(filepath.Join(tmpDir, rel))
		require.NoError(t, err, "output file %s must exist", rel)
		assert.Greater(t, fi.Size(), int64(0), "output file %s must be non-empty", rel)
	}
}

func TestOutputFilesUsePackageName(t *testing.T) {
	tmpDir := t.TempDir()
	pkg := "mypkg"
	require.NoError(t, run(schemaPath("chatapp.yaml"), tmpDir, pkg))

	goFile := filepath.Join(tmpDir, "go", pkg+".go")
	content, err := os.ReadFile(goFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "package mypkg", "generated Go file must declare package mypkg")

	cHeaderFile := filepath.Join(tmpDir, "c", pkg+".h")
	content, err = os.ReadFile(cHeaderFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "#ifndef MYPKG_H", "generated C header must use package name in include guard")
	assert.Contains(t, string(content), "#define MYPKG_H", "generated C header must use package name in include guard")

	cFile := filepath.Join(tmpDir, "c", pkg+".c")
	content, err = os.ReadFile(cFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "#include \"mypkg.h\"", "generated C file must include the correct header")
}

func TestOutputDirectoryCreatedAutomatically(t *testing.T) {
	// verifies the output directory tree when it doesn't already exist.
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "deep", "nested", "dir")
	require.NoError(t, run(schemaPath("chatapp.yaml"), outDir, "messages"))
	assert.DirExists(t, filepath.Join(outDir, "go"))
	assert.DirExists(t, filepath.Join(outDir, "c"))
}

func TestNonExistentInputFile(t *testing.T) {
	err := run("no_such_file.yaml", t.TempDir(), "messages")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse:", "error from missing file must begin with 'parse:'")
}

func TestInvalidPackageName(t *testing.T) {
	// The regex ^[a-zA-Z_]*$ — digits and hyphens are forbidden.
	cmd := exec.Command("go", "run", ".", "-i", schemaPath("chatapp.yaml"), "-o", t.TempDir(), "-p", "my-package")
	out, err := cmd.CombinedOutput()
	assert.Error(t, err, "invalid package name must cause a non-zero exit")
	assert.Contains(t, string(out), "invalid package name", "error message must mention 'invalid package name'")
}

func TestGeneratedGoCompiles(t *testing.T) {
	schemas := []struct {
		file string
		pkg  string
	}{
		{"padding_alignment.yaml", "padtest"},
		{"all_types.yaml", "alltypes"},
	}
	for _, s := range schemas {
		t.Run(strings.TrimSuffix(s.file, ".yaml"), func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			require.NoError(t, run(schemaPath(s.file), outDir, s.pkg))

			goSrc := filepath.Join(outDir, "go", strings.ToLower(s.pkg)+".go")
			content, err := os.ReadFile(goSrc)
			require.NoError(t, err)

			// Build in an isolated module so there are no dependency issues.
			buildDir := t.TempDir()
			goMod := []byte("module tempmod\n\ngo 1.22\n")
			require.NoError(t, os.WriteFile(filepath.Join(buildDir, "go.mod"), goMod, 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(buildDir, s.pkg+".go"), content, 0o644))

			cmd := exec.Command("go", "build", ".")
			cmd.Dir = buildDir
			out, err := cmd.CombinedOutput()
			assert.NoError(t, err,
				"generated Go for %s must compile:\n%s", s.file, string(out))
		})
	}
}

func TestGeneratedCCompiles(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not found in PATH, skipping C compilation tests")
	}

	schemas := []struct {
		file string
		pkg  string
	}{
		{"padding_alignment.yaml", "padtest"},
		{"all_types.yaml", "alltypes"},
	}
	for _, s := range schemas {
		t.Run(strings.TrimSuffix(s.file, ".yaml"), func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()
			require.NoError(t, run(schemaPath(s.file), outDir, s.pkg))

			cSrc := filepath.Join(outDir, "c", strings.ToLower(s.pkg)+".c")
			cHeader := filepath.Join(outDir, "c", strings.ToLower(s.pkg)+".h")

			cmd := exec.Command("gcc", "-c", cSrc, cHeader)
			cmd.Dir = outDir
			out, err := cmd.CombinedOutput()
			fmt.Printf("GCC output for %s:\n%s\n", s.file, string(out))
			assert.NoError(t, err, "generated C for %s must compile:\n%s", s.file, string(out))
		})
	}
}
