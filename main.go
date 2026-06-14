package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/avatar31/wireforge/internal/compiler"
	"github.com/avatar31/wireforge/internal/schema"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(inputFile, outputDir, packageName)
		},
	}

	rootCmd.Flags().StringVarP(&inputFile, "in", "i", "", "Input OpenAPI YAML schema file (required)")
	rootCmd.Flags().StringVarP(&outputDir, "out", "o", "./output", "Output directory for generated files")
	rootCmd.Flags().StringVarP(&packageName, "package", "p", "messages", "Go package name for generated code")

	_ = rootCmd.MarkFlagRequired("in")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(inputFile, outputDir, packageName string) error {
	s, err := schema.ParseFile(inputFile)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if len(s.Messages) == 0 {
		return fmt.Errorf("no message schemas found in %s", inputFile)
	}

	cs, err := compiler.Compile(s, packageName)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	fmt.Printf("Compiled Schema: %+v", cs)

	return nil
}
