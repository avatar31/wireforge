package main

import (
	"fmt"
	"os"
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

func run(inputFile, outputDir, packageName string) error {
	fmt.Printf("Parsing file: %s\n", inputFile)
	s, err := schema.ParseFile(inputFile)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if len(s.Messages) == 0 {
		return fmt.Errorf("no message schemas found in %s", inputFile)
	}

	cs := compiler.Compile(s, packageName)

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
	defer cHeaderFile.Close()

	if err := codegen.GenerateCHeader(cHeaderFile, cs); err != nil {
		return fmt.Errorf("generating C header: %w", err)
	}
	fmt.Printf("  generated: %s\n", cHeaderPath)

	cSourcePath := filepath.Join(cOutputDir, fmt.Sprintf("%s.c", lowerPackageName))
	cSourceFile, err := os.Create(cSourcePath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", cSourcePath, err)
	}
	defer cSourceFile.Close()

	if err := codegen.GenerateC(cSourceFile, cs); err != nil {
		return fmt.Errorf("generating C source: %w", err)
	}
	fmt.Printf("  generated: %s\n", cSourcePath)
	fmt.Printf("\nwireforge: successfully generated %d message type(s)\n", len(cs.Messages))
	return nil
}

// TODO's:
// - Add documentation for all packages and functions in the codegen and compiler packages
// - Add support for more complex types (arrays, nested objects)
// - C TODO's:
// 		- Add better error handling in C code. Like instead of returning -1,
// 		define error code in header template and return accordingly.
// - Go TODO's:
