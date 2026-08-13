// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avatar31/wireforge/internal/codegen"
	"github.com/avatar31/wireforge/internal/compiler"
	"github.com/avatar31/wireforge/internal/schema"
)

var (
	packageRegex = regexp.MustCompile(`^[a-zA-Z_]*$`)

	// https://clang.llvm.org/docs/ClangFormatStyleOptions.html
	cFormatStyle = `{
		BasedOnStyle: Google,
		IndentWidth: 4,
		ColumnLimit: 100,
		AlignConsecutiveAssignments: true,
		AlignConsecutiveMacros: true,
		AllowShortBlocksOnASingleLine: Always,
		AllowShortIfStatementsOnASingleLine: Always,
		AllowShortCaseLabelsOnASingleLine: true,
		BinPackArguments: false
	}`
)

func main() {
	var inputFile string
	var outputDir string
	var packageName string

	rootCmd := &cobra.Command{
		Use:   "wireforge",
		Short: "Schema-driven code generator for Go and C wire protocol structs",
		Long: "wireforge takes an OpenAPI YAML specification and produces a fully self-contained\n" +
			"Go file and matching C files (.h and .c) with proper memory alignment, big-endian\n" +
			"wire format serialization, and comprehensive safety checks.",
		SilenceUsage: true, // Don't show usage on error
		RunE: func(cmd *cobra.Command, args []string) error {
			if !packageRegex.MatchString(packageName) {
				return fmt.Errorf("invalid package name: %s. Must match regex: %s", packageName, packageRegex.String())
			}

			if proceed := checkClangFormat(); !proceed {
				return nil // User chose not to proceed without clang-format
			}

			return run(inputFile, outputDir, packageName)
		},
	}

	rootCmd.Flags().StringVarP(&inputFile, "in", "i", "", "Input OpenAPI YAML schema file (required)")
	rootCmd.Flags().StringVarP(&outputDir, "out", "o", "./out", "Output directory for generated files")
	rootCmd.Flags().StringVarP(&packageName, "package", "p", "messages", "Go package name for generated code")

	_ = rootCmd.MarkFlagRequired("in")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkClangFormat() bool {
	_, err := exec.LookPath("clang-format")
	if err == nil {
		return true
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("⚠️  'clang-format' was not found on your system. C Output code will NOT be styled.\n")
		fmt.Print("Do you want to continue anyway? (y/n): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input. Aborting.")
			return false
		}

		// Clean up the text input (remove spaces and line endings)
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "Y", "YES", "Yes", "yes":
			return true
		case "n", "N", "NO", "No", "no":
			return false
		default:
			fmt.Println("Invalid input. Please type 'y' or 'n'.")
		}
	}
}

func run(inputFile, outputDir, packageName string) error {
	fmt.Printf("Parsing file: %s\n", inputFile)
	s, err := schema.ParseFile(inputFile)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if len(s.Messages) == 0 {
		return fmt.Errorf("no message schemas found in %s", inputFile)
	}

	cs, err := compiler.Compile(s, packageName)
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
	}

	goOutputDir := filepath.Join(outputDir, "go")
	if err := os.MkdirAll(goOutputDir, 0o755); err != nil {
		return fmt.Errorf("creating Go output dir: %w", err)
	}

	lowerPackageName := strings.ToLower(packageName)
	goPath := filepath.Join(goOutputDir, fmt.Sprintf("%s.go", lowerPackageName))
	goFile, err := os.Create(goPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", goPath, err)
	}
	defer goFile.Close()

	if err := codegen.GenerateGo(goFile, cs); err != nil {
		return fmt.Errorf("generating Go code: %w", err)
	}
	fmt.Printf("  generated: %s\n", goPath)

	cOutputDir := filepath.Join(outputDir, "c")
	if err := os.MkdirAll(cOutputDir, 0o755); err != nil {
		return fmt.Errorf("creating C output dir: %w", err)
	}

	cHeaderPath := filepath.Join(cOutputDir, fmt.Sprintf("%s.h", lowerPackageName))
	cHeaderFile, err := os.Create(cHeaderPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", cHeaderPath, err)
	}

	if err := codegen.GenerateCHeader(cHeaderFile, cs); err != nil {
		cHeaderFile.Close()
		return fmt.Errorf("generating C header: %w", err)
	}
	cHeaderFile.Close()

	cHeaderFormatCmd := exec.Command("clang-format", "--style="+cFormatStyle, "-i", cHeaderPath)
	cHeaderFormatCmd.Stderr = os.Stderr
	err = cHeaderFormatCmd.Run()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to run clang-format on C Header file %s: %v\n", cHeaderPath, err)
	}

	fmt.Printf("  generated: %s\n", cHeaderPath)

	cSourcePath := filepath.Join(cOutputDir, fmt.Sprintf("%s.c", lowerPackageName))
	cSourceFile, err := os.Create(cSourcePath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", cSourcePath, err)
	}

	if err := codegen.GenerateC(cSourceFile, cs); err != nil {
		cSourceFile.Close()
		return fmt.Errorf("generating C source: %w", err)
	}
	cSourceFile.Close()
	cSourceFormatCmd := exec.Command("clang-format", "--style="+cFormatStyle, "-i", cSourcePath)
	cSourceFormatCmd.Stderr = os.Stderr
	err = cSourceFormatCmd.Run()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to run clang-format on C Source file %s: %v\n", cHeaderPath, err)
	}

	fmt.Printf("  generated: %s\n", cSourcePath)
	fmt.Printf("\nwireforge: successfully generated %d message type(s)\n", len(cs.Messages))

	return nil
}

// TODO's:
// - Add documentation for all packages and functions in the codegen and compiler packages
// - Test Padding logic after implementing more complex types like arrays and nested objects
// - C TODO's:
// 		- Add better error handling in C code. Like instead of returning -1,
// 		define error code in header template and return accordingly.
// - Go TODO's:
// 		- Add better error handling in Go code. Instead returning random error messages,
// 		define error types and return accordingly.
