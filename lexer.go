package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	src    []byte
	pos    int
	line   int
	col    int
	file   string
	tokens []Token
	errors []TranspileError
}

func NewLexer(src []byte, file string) *Lexer {
	return &Lexer{src: src, pos: 0, line: 1, col: 1, file: file}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.src[l.pos:])
	return r
}

func (l *Lexer) peek2() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	_, size := utf8.DecodeRune(l.src[l.pos:])
	if l.pos+size >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.src[l.pos+size:])
	return r
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, size := utf8.DecodeRune(l.src[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) makeToken(t TokenType, lexeme string, line, col int) Token {
	return Token{Type: t, Lexeme: lexeme, Line: line, Col: col, File: l.file}
}

func (l *Lexer) addError(line, col int, msg string, hints ...string) {
	l.errors = append(l.errors, TranspileError{
		Phase: "lexer", Severity: SEV_ERROR, File: l.file,
		Line: line, Col: col, Message: msg, Hints: hints,
	})
}

func (l *Lexer) Tokenize() ([]Token, []TranspileError) {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
		if tok.Type == TOKEN_INVALID {
			break
		}
	}
	return l.tokens, l.errors
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
			continue
		}
		if ch == '/' && l.peek2() == '/' {
			for l.pos < len(l.src) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		if ch == '/' && l.peek2() == '*' {
			startLine, startCol := l.line, l.col
			l.advance()
			l.advance()
			closed := false
			for l.pos < len(l.src) {
				c := l.advance()
				if c == '*' && l.peek() == '/' {
					l.advance()
					closed = true
					break
				}
			}
			if !closed {
				l.addError(startLine, startCol, "unclosed block comment /* ... */",
					"Add a closing */ to terminate the block comment")
			}
			continue
		}
		break
	}
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespaceAndComments()
	if l.pos >= len(l.src) {
		return l.makeToken(TOKEN_EOF, "", l.line, l.col)
	}

	line, col := l.line, l.col
	ch := l.peek()

	if ch == '@' {
		l.advance()
		if l.peek() == '"' {
			l.advance()
			return l.readVerbatimString(line, col)
		}
		return l.makeToken(TOKEN_AT, "@", line, col)
	}

	if unicode.IsLetter(ch) || ch == '_' {
		return l.readIdentOrKeyword(line, col)
	}

	if unicode.IsDigit(ch) || (ch == '.' && unicode.IsDigit(l.peek2())) {
		return l.readNumber(line, col)
	}

	if ch == '"' {
		l.advance()
		return l.readString(line, col)
	}

	if ch == '\'' {
		l.advance()
		return l.readChar(line, col)
	}

	return l.readOperator(line, col)
}

func (l *Lexer) readIdentOrKeyword(line, col int) Token {
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.peek()
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	word := sb.String()
	if tt, ok := keywords[word]; ok {
		return l.makeToken(tt, word, line, col)
	}
	return l.makeToken(TOKEN_IDENT, word, line, col)
}

func (l *Lexer) readNumber(line, col int) Token {
	var sb strings.Builder
	isFloat := false

	if l.peek() == '0' && (l.peek2() == 'x' || l.peek2() == 'X') {
		sb.WriteRune(l.advance())
		sb.WriteRune(l.advance())
		for l.pos < len(l.src) {
			c := l.peek()
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '_' {
				sb.WriteRune(l.advance())
			} else {
				break
			}
		}
		if p := l.peek(); p == 'L' || p == 'l' || p == 'U' || p == 'u' {
			sb.WriteRune(l.advance())
		}
		return l.makeToken(TOKEN_INT_LIT, sb.String(), line, col)
	}

	for l.pos < len(l.src) && (unicode.IsDigit(l.peek()) || l.peek() == '_') {
		sb.WriteRune(l.advance())
	}

	if l.peek() == '.' && unicode.IsDigit(l.peek2()) {
		isFloat = true
		sb.WriteRune(l.advance())
		for l.pos < len(l.src) && (unicode.IsDigit(l.peek()) || l.peek() == '_') {
			sb.WriteRune(l.advance())
		}
	}

	if p := l.peek(); p == 'e' || p == 'E' {
		isFloat = true
		sb.WriteRune(l.advance())
		if p2 := l.peek(); p2 == '+' || p2 == '-' {
			sb.WriteRune(l.advance())
		}
		for l.pos < len(l.src) && unicode.IsDigit(l.peek()) {
			sb.WriteRune(l.advance())
		}
	}

	suffix := l.peek()
	switch {
	case suffix == 'f' || suffix == 'F':
		sb.WriteRune(l.advance())
		return l.makeToken(TOKEN_FLOAT_LIT, sb.String(), line, col)
	case suffix == 'd' || suffix == 'D':
		sb.WriteRune(l.advance())
		return l.makeToken(TOKEN_DOUBLE_LIT, sb.String(), line, col)
	case suffix == 'm' || suffix == 'M':
		l.addError(line, col, "decimal literals (suffix 'm') are not supported",
			"Use double instead: remove the 'm' suffix and change the type to double")
		sb.WriteRune(l.advance())
		return l.makeToken(TOKEN_INVALID, sb.String(), line, col)
	case suffix == 'L' || suffix == 'l' || suffix == 'U' || suffix == 'u':
		sb.WriteRune(l.advance())
		return l.makeToken(TOKEN_INT_LIT, sb.String(), line, col)
	}

	if isFloat {
		return l.makeToken(TOKEN_DOUBLE_LIT, sb.String(), line, col)
	}
	return l.makeToken(TOKEN_INT_LIT, sb.String(), line, col)
}

func (l *Lexer) readString(line, col int) Token {
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			return l.makeToken(TOKEN_STRING_LIT, sb.String(), line, col)
		}
		if ch == '\n' || ch == 0 {
			l.addError(line, col, "unterminated string literal",
				"Add a closing '\"' to end the string",
				"For multi-line strings, use verbatim strings: @\"...\"")
			break
		}
		if ch == '\\' {
			l.advance()
			esc := l.advance()
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '0':
				sb.WriteByte(0)
			case 'a':
				sb.WriteByte('\a')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'v':
				sb.WriteByte('\v')
			default:
				l.addError(l.line, l.col-1, fmt.Sprintf("unknown escape sequence '\\%c'", esc),
					"Valid escapes: \\n \\t \\r \\\\ \\\" \\0 \\a \\b \\f \\v")
				sb.WriteRune('\\')
				sb.WriteRune(esc)
			}
		} else {
			sb.WriteRune(l.advance())
		}
	}
	return l.makeToken(TOKEN_STRING_LIT, sb.String(), line, col)
}

func (l *Lexer) readVerbatimString(line, col int) Token {
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			if l.peek() == '"' {
				sb.WriteByte('"')
				l.advance()
			} else {
				return l.makeToken(TOKEN_STRING_LIT, sb.String(), line, col)
			}
		} else {
			sb.WriteRune(l.advance())
		}
	}
	l.addError(line, col, "unterminated verbatim string literal",
		"Add a closing '\"' to end the verbatim string")
	return l.makeToken(TOKEN_STRING_LIT, sb.String(), line, col)
}

func (l *Lexer) readChar(line, col int) Token {
	var ch rune
	if l.peek() == '\\' {
		l.advance()
		esc := l.advance()
		switch esc {
		case 'n':
			ch = '\n'
		case 't':
			ch = '\t'
		case 'r':
			ch = '\r'
		case '\\':
			ch = '\\'
		case '\'':
			ch = '\''
		case '0':
			ch = 0
		default:
			l.addError(line, col, fmt.Sprintf("unknown char escape '\\%c'", esc),
				"Valid char escapes: \\n \\t \\r \\\\ \\' \\0")
			ch = esc
		}
	} else {
		ch = l.advance()
	}
	if l.peek() != '\'' {
		l.addError(line, col, "char literal not properly closed with \"'\"",
			"Char literals can only contain a single character",
			"For multi-char sequences, use a string: \"...\"")
	} else {
		l.advance()
	}
	return l.makeToken(TOKEN_CHAR_LIT, string(ch), line, col)
}

func (l *Lexer) readOperator(line, col int) Token {
	ch := l.advance()
	switch ch {
	case '+':
		if l.peek() == '+' {
			l.advance()
			return l.makeToken(TOKEN_PLUS_PLUS, "++", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_PLUS_EQ, "+=", line, col)
		}
		return l.makeToken(TOKEN_PLUS, "+", line, col)
	case '-':
		if l.peek() == '-' {
			l.advance()
			return l.makeToken(TOKEN_MINUS_MINUS, "--", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_MINUS_EQ, "-=", line, col)
		} else if l.peek() == '>' {
			l.advance()
			return l.makeToken(TOKEN_ARROW, "->", line, col)
		}
		return l.makeToken(TOKEN_MINUS, "-", line, col)
	case '*':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_STAR_EQ, "*=", line, col)
		}
		return l.makeToken(TOKEN_STAR, "*", line, col)
	case '/':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_SLASH_EQ, "/=", line, col)
		}
		return l.makeToken(TOKEN_SLASH, "/", line, col)
	case '%':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_PERCENT_EQ, "%=", line, col)
		}
		return l.makeToken(TOKEN_PERCENT, "%", line, col)
	case '&':
		if l.peek() == '&' {
			l.advance()
			return l.makeToken(TOKEN_AMP_AMP, "&&", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_AMP_EQ, "&=", line, col)
		}
		return l.makeToken(TOKEN_AMP, "&", line, col)
	case '|':
		if l.peek() == '|' {
			l.advance()
			return l.makeToken(TOKEN_PIPE_PIPE, "||", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_PIPE_EQ, "|=", line, col)
		}
		return l.makeToken(TOKEN_PIPE, "|", line, col)
	case '^':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_CARET_EQ, "^=", line, col)
		}
		return l.makeToken(TOKEN_CARET, "^", line, col)
	case '~':
		return l.makeToken(TOKEN_TILDE, "~", line, col)
	case '!':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_BANG_EQ, "!=", line, col)
		}
		return l.makeToken(TOKEN_BANG, "!", line, col)
	case '<':
		if l.peek() == '<' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return l.makeToken(TOKEN_LSHIFT_EQ, "<<=", line, col)
			}
			return l.makeToken(TOKEN_LSHIFT, "<<", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_LT_EQ, "<=", line, col)
		}
		return l.makeToken(TOKEN_LT, "<", line, col)
	case '>':
		if l.peek() == '>' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return l.makeToken(TOKEN_RSHIFT_EQ, ">>=", line, col)
			}
			return l.makeToken(TOKEN_RSHIFT, ">>", line, col)
		} else if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_GT_EQ, ">=", line, col)
		}
		return l.makeToken(TOKEN_GT, ">", line, col)
	case '=':
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TOKEN_EQ_EQ, "==", line, col)
		}
		return l.makeToken(TOKEN_EQ, "=", line, col)
	case '?':
		if l.peek() == '?' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return l.makeToken(TOKEN_NULL_ASGN, "??=", line, col)
			}
			return l.makeToken(TOKEN_NULL_COAL, "??", line, col)
		}
		return l.makeToken(TOKEN_QUESTION, "?", line, col)
	case ':':
		if l.peek() == ':' {
			l.advance()
			return l.makeToken(TOKEN_COLON_COLON, "::", line, col)
		}
		return l.makeToken(TOKEN_COLON, ":", line, col)
	case '.':
		return l.makeToken(TOKEN_DOT, ".", line, col)
	case ',':
		return l.makeToken(TOKEN_COMMA, ",", line, col)
	case ';':
		return l.makeToken(TOKEN_SEMICOLON, ";", line, col)
	case '(':
		return l.makeToken(TOKEN_LPAREN, "(", line, col)
	case ')':
		return l.makeToken(TOKEN_RPAREN, ")", line, col)
	case '{':
		return l.makeToken(TOKEN_LBRACE, "{", line, col)
	case '}':
		return l.makeToken(TOKEN_RBRACE, "}", line, col)
	case '[':
		return l.makeToken(TOKEN_LBRACKET, "[", line, col)
	case ']':
		return l.makeToken(TOKEN_RBRACKET, "]", line, col)
	}

	l.addError(line, col, fmt.Sprintf("unexpected character '%c' (U+%04X)", ch, ch),
		"Remove this character or replace it with a valid token")
	return l.makeToken(TOKEN_INVALID, string(ch), line, col)
}
