//go:build generate

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:generate go run gen.go -type=BlockSize,CompressionLevel -output ../options_gen.go

type BlockSize int

const (
	Small BlockSize = iota
	Medium
	Large
)

type CompressionLevel int

const (
	Default CompressionLevel = iota
	BestSpeed
	BestCompression
)

// GenerateStringer executes the 'stringer' tool for the specified types.
// It replicates running `go run golang.org/x/tools/cmd/stringer -type=<types> -output=<outputRelPath>`
// within the 'packageSourceDir'.
//
// Args:
//   - types: A slice of type names (e.g., []string{"BlockSize", "CompressionLevel"}).
//   - outputRelPath: The path for the generated .go file. This path is relative
//     to the packageSourceDir. For example, if packageSourceDir is "./models" and
//     outputRelPath is "models_string.go", the output will be "./models/models_string.go".
//     If outputRelPath is "../options_gen.go" and packageSourceDir is "./cmd/app",
//     the output will be "./cmd/options_gen.go".
//   - packageSourceDir: The directory containing the Go source files where the
//     types are defined. Stringer will be executed with this directory as its
//     working directory.
func GenerateStringer(types []string, outputRelPath string, packageSourceDir string) error {
	if len(types) == 0 {
		return fmt.Errorf("no types provided for stringer")
	}
	if packageSourceDir == "" {
		return fmt.Errorf("package source directory must be specified")
	}
	if outputRelPath == "" {
		return fmt.Errorf("output file path must be specified")
	}

	// Ensure packageSourceDir exists and is a directory
	info, err := os.Stat(packageSourceDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("package source directory %s does not exist: %w", packageSourceDir, err)
	}
	if err != nil {
		return fmt.Errorf("error stating package source directory %s: %w", packageSourceDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("package source directory %s is not a directory", packageSourceDir)
	}

	typeArg := strings.Join(types, ",")

	// The stringer tool's -output flag creates a file relative to its working directory (cmd.Dir)
	cmd := exec.Command("go", "run", "golang.org/x/tools/cmd/stringer", "-type", typeArg, "-output", outputRelPath)
	cmd.Dir = packageSourceDir // Set the working directory for stringer

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	log.Printf("Executing stringer in directory '%s': go run golang.org/x/tools/cmd/stringer -type %s -output %s\n",
		packageSourceDir, typeArg, outputRelPath)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("stringer command failed: %w\nStdout: %s\nStderr: %s",
			err, stdoutBuf.String(), stderrBuf.String())
	}

	fullOutputPath := filepath.Join(packageSourceDir, outputRelPath)
	log.Printf("Stringer successfully generated: %s\nStdout: %s\nStderr: %s",
		fullOutputPath, stdoutBuf.String(), stderrBuf.String())
	return nil
}

func main() {
	// --- Example of calling GenerateStringer internally ---
	// This section demonstrates how to use the GenerateStringer function.
	// It's configured to mimic the behavior of the `//go:generate` directive above.

	// 1. Determine the source directory of the package containing the types.
	//    The `go:generate` directive executes stringer in the context of the directory
	//    containing the file with the `//go:generate` comment.
	_, currentFilePath, _, ok := runtime.Caller(0) // Gets the path of this main.go file
	if !ok {
		log.Fatal("Error: Could not determine current file path to establish package source directory.")
	}
	// packageDirForGenerate is the directory where this main.go file resides.
	// This is the directory stringer would use as its context for the go:generate line.
	packageDirForGenerate := filepath.Dir(currentFilePath)

	// 2. Define the types to process.
	typesToProcess := []string{"BlockSize", "CompressionLevel"}

	// 3. Define the output path, relative to packageDirForGenerate.
	//    This matches the `-output ../options_gen.go` from the directive.
	outputFileRelPath := "../options_gen.go"

	fmt.Printf("Demonstrating internal call to GenerateStringer:\n")
	fmt.Printf("  Package source directory (context for stringer): %s\n", packageDirForGenerate)
	fmt.Printf("  Types to process: %v\n", typesToProcess)
	fmt.Printf("  Output path (relative to source directory): %s\n", outputFileRelPath)
	absOutputPath := filepath.Clean(filepath.Join(packageDirForGenerate, outputFileRelPath))
	fmt.Printf("  Expected absolute output path: %s\n", absOutputPath)

	// Note: For this to run without errors, you must have:
	//   - Go toolchain installed.
	//   - The `golang.org/x/tools/cmd/stringer` package accessible.
	//   - The types (BlockSize, CompressionLevel) defined in .go files within `packageDirForGenerate`.
	//     (See commented out `types.go` example above).

	// Uncomment the following block to actually execute the stringer generation.
	/*
		err := GenerateStringer(typesToProcess, outputFileRelPath, packageDirForGenerate)
		if err != nil {
			log.Fatalf("Error calling GenerateStringer internally: %v", err)
		}
		fmt.Println("Internal call to GenerateStringer completed successfully.")
		fmt.Printf("Generated file should be at: %s\n", absOutputPath)
	*/

	fmt.Println("\nProgram finished. Uncomment the GenerateStringer call in main() to run it.")
	fmt.Println("Remember to define the 'BlockSize' and 'CompressionLevel' types in this package for it to work.")
}
