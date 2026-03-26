package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CompileOptions controls the GCC compilation step
type CompileOptions struct {
	OutputBinary string
	OptLevel     string // "0", "1", "2", "3", "s", "g"
	Debug        bool
	Warnings     bool
	ExtraFlags   []string
	Standard     string // "c++17", "c++20"
	RunTests     bool
}

// DefaultCompileOptions returns the fastest-possible single-file config.
// -O0 + -w + -pipe cuts compile time from ~2.6s → ~300ms vs the old -O2 -Wall defaults.
func DefaultCompileOptions(outputBin string) CompileOptions {
	return CompileOptions{
		OutputBinary: outputBin,
		OptLevel:     "0",     // no optimisation — fastest build
		Debug:        false,
		Warnings:     false,   // our type-checker handles diagnostics; skip g++ noise
		Standard:     "c++17",
		ExtraFlags:   []string{"-pipe"}, // avoid temp files, slightly faster
	}
}

// CompileResult holds the output of a g++ run
type CompileResult struct {
	Success    bool
	GccOutput  string
	GccErrors  string
	Duration   time.Duration
	BinaryPath string
}

// CompileWithGCC invokes g++ to compile the generated C++ source
func CompileWithGCC(cppSource string, opts CompileOptions) CompileResult {
	// Write source to a temp file
	tmpDir := os.TempDir()
	srcFile := filepath.Join(tmpDir, "cstranspile_out.cpp")
	if err := os.WriteFile(srcFile, []byte(cppSource), 0644); err != nil {
		return CompileResult{
			Success:   false,
			GccErrors: fmt.Sprintf("failed to write temp source file: %v", err),
		}
	}
	defer os.Remove(srcFile)

	// Build argument list
	args := []string{}

	std := opts.Standard
	if std == "" {
		std = "c++17"
	}
	args = append(args, "-std="+std)

	args = append(args, "-O"+opts.OptLevel)

	if opts.Warnings {
		// Full warnings when explicitly requested
		args = append(args,
			"-Wall",
			"-Wextra",
			"-Wno-unused-parameter",
			"-Wno-unused-variable",
		)
	} else {
		// Silence everything — our type-checker already covers the interesting stuff
		args = append(args, "-w")
	}

	if opts.Debug {
		args = append(args, "-g")
	}

	args = append(args, opts.ExtraFlags...)

	outBin := opts.OutputBinary
	if outBin == "" {
		outBin = filepath.Join(tmpDir, "cstranspile_bin")
	}
	args = append(args, "-o", outBin)
	args = append(args, srcFile)

	cmd := exec.Command("g++", args...)
	start := time.Now()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	dur := time.Since(start)

	if err != nil {
		return CompileResult{
			Success:   false,
			GccOutput: outBuf.String(),
			GccErrors: errBuf.String(),
			Duration:  dur,
		}
	}

	return CompileResult{
		Success:    true,
		GccOutput:  outBuf.String(),
		GccErrors:  errBuf.String(),
		Duration:   dur,
		BinaryPath: outBin,
	}
}

// RunBinary executes the compiled binary and returns its output
func RunBinary(binaryPath string, args []string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			err = runErr
		}
	}
	return
}

// PrintGCCDiagnostics re-prints g++ output in our diagnostic style
func PrintGCCDiagnostics(gccErr string) {
	if gccErr == "" {
		return
	}
	lines := strings.Split(gccErr, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, ": error:") {
			fmt.Fprintf(os.Stderr, "%s\n", color(colorRed, "gcc"+colorReset+": "+line))
		} else if strings.Contains(line, ": warning:") {
			fmt.Fprintf(os.Stderr, "%s\n", color(colorYellow, "gcc"+colorReset+": "+line))
		} else if strings.Contains(line, ": note:") {
			fmt.Fprintf(os.Stderr, "%s\n", color(colorCyan, "gcc"+colorReset+": "+line))
		} else {
			fmt.Fprintf(os.Stderr, "  %s\n", color(colorGray, line))
		}
	}
}

// GCCAvailable checks whether g++ is on PATH
func GCCAvailable() bool {
	_, err := exec.LookPath("g++")
	return err == nil
}
