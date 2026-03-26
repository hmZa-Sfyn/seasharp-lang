package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "0.1.0"

const helpText = `
╔══════════════════════════════════════════════════════════════╗
║         csx — C# → C++ Transpiler  v` + version + `         ║
╚══════════════════════════════════════════════════════════════╝

USAGE:
    csx [options] <input.cs>

OPTIONS:
    -o <file>       Output binary path (default: ./a.out)
    -O <level>      GCC optimization level: 0,1,2,3,s,g (default: 2)
    -debug          Compile with debug symbols (-g)
    -emit-cpp       Only emit C++ source, do not compile
    -cpp-out <f>    Write C++ source to file (default: stdout if -emit-cpp)
    -test           Run @test-annotated methods after compilation
    -no-warn        Suppress transpiler warnings
    -no-color       Disable colored output
    -std <std>      C++ standard: c++17, c++20 (default: c++17)
    -v              Show version and exit
    -h / -help      Show this help

ANNOTATIONS:
    @test           Mark a void method as a unit test (runs with -test flag)
    @deprecated     Mark a member as deprecated (emits warning)
    @inline         Hint to compiler to inline a method

EXAMPLES:
    csx program.cs
    csx -o myapp -O3 program.cs
    csx -test program.cs
    csx -emit-cpp -cpp-out out.cpp program.cs

C# FEATURES SUPPORTED:
    • Classes, structs, interfaces, enums (with inheritance)
    • Generics (mapped to C++ templates/STL)
    • Namespaces (mapped to C++ namespaces)
    • Properties with get/set (mapped to getter/setter methods)
    • All C# operators including ??, ??=, ternary
    • foreach, for, while, do-while, switch
    • try/catch/finally
    • ref/out parameters
    • Nullable types (T?)
    • Arrays → std::vector
    • string → std::string
    • Console.WriteLine → std::cout
    • Math.* → std::*
    • typeof() → typeid().name()
    • Type casts (int)x → static_cast<int>(x)
    • is/as → dynamic_cast

`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		os.Exit(1)
	}

	var (
		outputBin = flag.String("o", "a.out", "Output binary path")
		optLevel  = flag.String("O", "2", "GCC optimization level")
		debug     = flag.Bool("debug", false, "Compile with debug symbols")
		emitCpp   = flag.Bool("emit-cpp", false, "Only emit C++ source")
		cppOut    = flag.String("cpp-out", "", "Write C++ source to file")
		runTests  = flag.Bool("test", false, "Run @test methods")
		noWarn    = flag.Bool("no-warn", false, "Suppress warnings")
		noColor   = flag.Bool("no-color", false, "Disable colored output")
		cppStd    = flag.String("std", "c++17", "C++ standard")
		showVer   = flag.Bool("v", false, "Show version")
		_         = flag.Bool("h", false, "Show help")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("csx v%s\n", version)
		os.Exit(0)
	}

	if *noColor {
		useColor = false
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Print(helpText)
		os.Exit(1)
	}

	inputFile := args[0]

	// ── Read source ───────────────────────────────────────────────────────────
	startTotal := time.Now()

	src, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot open '%s': %v\n",
			color(colorRed, "error"), inputFile, err)
		os.Exit(1)
	}

	srcLines := strings.Split(string(src), "\n")
	srcMap := map[string][]string{inputFile: srcLines}

	var allErrors []TranspileError

	printPhaseHeader("Lexing")
	lexStart := time.Now()
	lexer := NewLexer(src, inputFile)
	tokens, lexErrs := lexer.Tokenize()
	allErrors = append(allErrors, lexErrs...)
	lexDur := time.Since(lexStart)
	printPhaseDone("Lexer", lexDur, len(tokens), "tokens", len(lexErrs))

	if hasErrors(lexErrs) {
		PrintDiagnostics(allErrors, srcMap)
		PrintSummary(allErrors)
		os.Exit(1)
	}

	// ── Parse ─────────────────────────────────────────────────────────────────
	printPhaseHeader("Parsing")
	parseStart := time.Now()
	parser := NewParser(tokens, inputFile)
	cu, parseErrs := parser.ParseFile()
	allErrors = append(allErrors, parseErrs...)
	parseDur := time.Since(parseStart)
	printPhaseDone("Parser", parseDur, countDecls(cu), "declarations", len(parseErrs))

	if hasErrors(parseErrs) {
		PrintDiagnostics(allErrors, srcMap)
		PrintSummary(allErrors)
		os.Exit(1)
	}

	// ── Type Check ────────────────────────────────────────────────────────────
	printPhaseHeader("Type Checking")
	tcStart := time.Now()
	tc := NewTypeChecker(inputFile)
	tcErrs := tc.Check(cu)
	// Filter warnings if requested
	if *noWarn {
		var filtered []TranspileError
		for _, e := range tcErrs {
			if e.Severity == SEV_ERROR {
				filtered = append(filtered, e)
			}
		}
		tcErrs = filtered
	}
	allErrors = append(allErrors, tcErrs...)
	tcDur := time.Since(tcStart)
	errCount := 0
	for _, e := range tcErrs {
		if e.Severity == SEV_ERROR {
			errCount++
		}
	}
	printPhaseDone("TypeChecker", tcDur, len(tcErrs), "diagnostics", errCount)

	// Print all diagnostics so far (non-fatal warnings shown too)
	PrintDiagnostics(allErrors, srcMap)

	if hasErrors(tcErrs) {
		PrintSummary(allErrors)
		os.Exit(1)
	}

	// ── Emit C++ ──────────────────────────────────────────────────────────────
	printPhaseHeader("Emitting C++")
	emitStart := time.Now()
	emitter := NewEmitter(inputFile)
	cppCode, emitErrs := emitter.Emit(cu)
	allErrors = append(allErrors, emitErrs...)
	emitDur := time.Since(emitStart)
	printPhaseDone("Emitter", emitDur, strings.Count(cppCode, "\n"), "lines", len(emitErrs))

	if hasErrors(emitErrs) {
		PrintDiagnostics(emitErrs, srcMap)
		PrintSummary(allErrors)
		os.Exit(1)
	}

	testMethods := emitter.GetTestMethods()

	// Add assert header + test runner if needed
	if *runTests && len(testMethods) > 0 {
		cppCode = cppCode + GenerateAssertHeader() + GenerateTestRunner(testMethods)
	} else if len(testMethods) > 0 && !*runTests {
		fmt.Fprintf(os.Stderr, "%s found %d @test method(s) — use -test to run them\n",
			color(colorCyan, "note:"), len(testMethods))
	}

	// ── Write or compile ──────────────────────────────────────────────────────
	if *emitCpp {
		if *cppOut != "" {
			if err := os.WriteFile(*cppOut, []byte(cppCode), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: cannot write '%s': %v\n",
					color(colorRed, "error"), *cppOut, err)
				os.Exit(1)
			}
			fmt.Printf("%s wrote C++ source to %s\n",
				color(colorGreen, "ok:"), *cppOut)
		} else {
			fmt.Print(cppCode)
		}
		printTotal(time.Since(startTotal))
		return
	}

	// Save cpp to file alongside input if -cpp-out specified
	if *cppOut != "" {
		if err := os.WriteFile(*cppOut, []byte(cppCode), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write cpp source: %v\n",
				color(colorRed, "error"), err)
		}
	}

	// ── Compile with GCC ──────────────────────────────────────────────────────
	if !GCCAvailable() {
		fmt.Fprintf(os.Stderr, "%s: g++ not found in PATH\n", color(colorRed, "error"))
		fmt.Fprintf(os.Stderr, "  %s: install g++ with: sudo apt-get install g++\n",
			color(colorGreen, "hint"))
		os.Exit(1)
	}

	printPhaseHeader("Compiling with g++")
	opts := CompileOptions{
		OutputBinary: *outputBin,
		OptLevel:     *optLevel,
		Debug:        *debug,
		Warnings:     true,
		Standard:     *cppStd,
	}

	result := CompileWithGCC(cppCode, opts)

	if !result.Success {
		fmt.Fprintf(os.Stderr, "\n%s: g++ compilation failed (%.2fs)\n\n",
			color(colorRed, "error"), result.Duration.Seconds())
		PrintGCCDiagnostics(result.GccErrors)
		os.Exit(1)
	}

	fmt.Printf("%s compiled '%s' → '%s' in %.2fs  (-%s optimizations)\n",
		color(colorGreen, "ok:"),
		filepath.Base(inputFile),
		*outputBin,
		result.Duration.Seconds(),
		*optLevel)

	if result.GccErrors != "" {
		// gcc warnings
		PrintGCCDiagnostics(result.GccErrors)
	}

	// ── Run tests if requested ────────────────────────────────────────────────
	if *runTests {
		if len(testMethods) == 0 {
			fmt.Printf("%s no @test methods found\n", color(colorYellow, "warning:"))
		} else {
			fmt.Printf("\n%s running %d test(s)...\n\n",
				color(colorCyan, "test:"), len(testMethods))
			stdout, stderr, code, _ := RunBinary(*outputBin, nil)
			fmt.Print(stdout)
			if stderr != "" {
				fmt.Fprint(os.Stderr, stderr)
			}
			if code != 0 {
				fmt.Fprintf(os.Stderr, "\n%s tests failed (exit code %d)\n",
					color(colorRed, "FAIL:"), code)
				printTotal(time.Since(startTotal))
				os.Exit(code)
			} else {
				fmt.Printf("\n%s all tests passed\n", color(colorGreen, "PASS:"))
			}
		}
	}

	printTotal(time.Since(startTotal))
	PrintSummary(allErrors)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printPhaseHeader(name string) {
	fmt.Printf("%s %s\n",
		color(colorBlue, "  -->"),
		color(colorBold, name+"..."))
}

func printPhaseDone(phase string, dur time.Duration, count int, unit string, errs int) {
	errStr := ""
	if errs > 0 {
		errStr = fmt.Sprintf("  %s", color(colorRed, fmt.Sprintf("%d error(s)", errs)))
	}
	fmt.Printf("       %s %s%s: %d %s in %s%s\n",
		color(colorGreen, "✓"),
		color(colorGray, phase),
		"",
		count, unit,
		color(colorGray, dur.String()),
		errStr,
	)
}

func printTotal(dur time.Duration) {
	fmt.Printf("\n%s finished in %s\n",
		color(colorGreen, "done:"),
		color(colorBold, dur.Round(time.Millisecond).String()))
}

func countDecls(cu *CompilationUnit) int {
	count := len(cu.Members)
	for _, ns := range cu.Namespaces {
		count += countNsDecls(ns)
	}
	return count
}

func countNsDecls(ns *NamespaceDecl) int {
	count := len(ns.Members)
	for _, d := range ns.Members {
		if nested, ok := d.(*NamespaceDecl); ok {
			count += countNsDecls(nested)
		}
	}
	return count
}
