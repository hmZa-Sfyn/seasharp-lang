package main

import (
	"fmt"
	"os"
	"strings"
)

type Severity int

const (
	SEV_ERROR Severity = iota
	SEV_WARNING
	SEV_NOTE
)

type TranspileError struct {
	Phase    string
	Severity Severity
	File     string
	Line     int
	Col      int
	Message  string
	Hints    []string
	Note     string
	Code     string
}

func (e *TranspileError) Error() string { return e.Message }

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[1;31m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[1;36m"
	colorBlue   = "\033[1;34m"
	colorGreen  = "\033[1;32m"
	colorGray   = "\033[0;37m"
	colorBold   = "\033[1m"
	colorPurple = "\033[1;35m"
)

var useColor = isTerminal()

func isTerminal() bool {
	fi, _ := os.Stderr.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func color(c, s string) string {
	if !useColor {
		return s
	}
	return c + s + colorReset
}

func PrintDiagnostics(errs []TranspileError, src map[string][]string) {
	for i := range errs {
		printDiag(&errs[i], src)
	}
}

func printDiag(e *TranspileError, src map[string][]string) {
	var sb strings.Builder

	prefix := ""
	switch e.Severity {
	case SEV_ERROR:
		prefix = color(colorRed, "error")
	case SEV_WARNING:
		prefix = color(colorYellow, "warning")
	case SEV_NOTE:
		prefix = color(colorCyan, "note")
	}
	if e.Code != "" {
		prefix += color(colorRed, "["+e.Code+"]")
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", prefix, color(colorBold, e.Message)))

	if e.File != "" || e.Line > 0 {
		loc := ""
		if e.File != "" {
			loc = e.File
		}
		if e.Line > 0 {
			if loc != "" {
				loc += fmt.Sprintf(":%d:%d", e.Line, e.Col)
			} else {
				loc = fmt.Sprintf("%d:%d", e.Line, e.Col)
			}
		}
		sb.WriteString(fmt.Sprintf("  %s %s\n", color(colorBlue, "-->"), color(colorGray, loc)))
	}

	if lines, ok := src[e.File]; ok && e.Line > 0 && e.Line <= len(lines) {
		lineNum := fmt.Sprintf("%d", e.Line)
		gutter := strings.Repeat(" ", len(lineNum))
		sourceLine := lines[e.Line-1]

		sb.WriteString(fmt.Sprintf("  %s\n", color(colorBlue, gutter+" |")))
		sb.WriteString(fmt.Sprintf("  %s %s\n", color(colorBlue, lineNum+" |"), sourceLine))

		col := e.Col - 1
		if col < 0 {
			col = 0
		}
		if col > len(sourceLine) {
			col = len(sourceLine)
		}
		caretLen := 1
		if e.Col > 0 && e.Col <= len(sourceLine) {
			// Try to highlight the whole token
			i := e.Col - 1
			for i < len(sourceLine) && sourceLine[i] != ' ' && sourceLine[i] != '(' && sourceLine[i] != ')' {
				i++
				caretLen++
			}
			if caretLen > 20 {
				caretLen = 1
			}
		}
		caretLine := strings.Repeat(" ", col) + color(colorRed, strings.Repeat("^", caretLen))
		sb.WriteString(fmt.Sprintf("  %s %s\n", color(colorBlue, gutter+" |"), caretLine))
	}

	if e.Phase != "" {
		sb.WriteString(fmt.Sprintf("  %s in phase: %s\n",
			color(colorGray, "="), color(colorGray, e.Phase)))
	}

	for _, h := range e.Hints {
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n",
			color(colorGreen, "="), color(colorGreen, "hint"), h))
	}

	if e.Note != "" {
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n",
			color(colorCyan, "="), color(colorCyan, "note"), e.Note))
	}

	sb.WriteString("\n")
	fmt.Fprint(os.Stderr, sb.String())
}

func PrintSummary(errs []TranspileError) {
	errCount, warnCount := 0, 0
	for _, e := range errs {
		switch e.Severity {
		case SEV_ERROR:
			errCount++
		case SEV_WARNING:
			warnCount++
		}
	}
	if errCount > 0 {
		s := ""
		if errCount > 1 {
			s = "s"
		}
		fmt.Fprintf(os.Stderr, "%s: aborting due to %s previous error%s\n",
			color(colorRed, "error"), color(colorBold, fmt.Sprintf("%d", errCount)), s)
	}
	if warnCount > 0 {
		s := ""
		if warnCount > 1 {
			s = "s"
		}
		fmt.Fprintf(os.Stderr, "%s: %d warning%s emitted\n",
			color(colorYellow, "warning"), warnCount, s)
	}
}

func newError(phase, file string, tok Token, code, msg string, hints ...string) TranspileError {
	line, col := tok.Line, tok.Col
	if line == 0 {
		line = 1
	}
	return TranspileError{Phase: phase, Severity: SEV_ERROR, File: file,
		Line: line, Col: col, Message: msg, Hints: hints, Code: code}
}

func newWarning(phase, file string, tok Token, code, msg string, hints ...string) TranspileError {
	line, col := tok.Line, tok.Col
	if line == 0 {
		line = 1
	}
	return TranspileError{Phase: phase, Severity: SEV_WARNING, File: file,
		Line: line, Col: col, Message: msg, Hints: hints, Code: code}
}

func newNote(phase, file string, tok Token, msg string) TranspileError {
	return TranspileError{Phase: phase, Severity: SEV_NOTE, File: file,
		Line: tok.Line, Col: tok.Col, Message: msg}
}

func hasErrors(errs []TranspileError) bool {
	for _, e := range errs {
		if e.Severity == SEV_ERROR {
			return true
		}
	}
	return false
}
