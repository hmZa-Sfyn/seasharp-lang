package main

import "fmt"

// TokenType represents all token kinds in the C#-like language
type TokenType int

const (
	// Literals
	TOKEN_INT_LIT    TokenType = iota // 123
	TOKEN_FLOAT_LIT                   // 3.14f
	TOKEN_DOUBLE_LIT                  // 3.14
	TOKEN_STRING_LIT                  // "hello"
	TOKEN_CHAR_LIT                    // 'a'
	TOKEN_BOOL_LIT                    // true/false
	TOKEN_NULL                        // null

	// Identifiers
	TOKEN_IDENT // myVar

	// Keywords
	TOKEN_NAMESPACE
	TOKEN_CLASS
	TOKEN_STRUCT
	TOKEN_INTERFACE
	TOKEN_ENUM
	TOKEN_USING
	TOKEN_IF
	TOKEN_ELSE
	TOKEN_WHILE
	TOKEN_FOR
	TOKEN_FOREACH
	TOKEN_IN
	TOKEN_DO
	TOKEN_SWITCH
	TOKEN_CASE
	TOKEN_DEFAULT
	TOKEN_BREAK
	TOKEN_CONTINUE
	TOKEN_RETURN
	TOKEN_NEW
	TOKEN_THIS
	TOKEN_BASE
	TOKEN_STATIC
	TOKEN_READONLY
	TOKEN_CONST
	TOKEN_PUBLIC
	TOKEN_PRIVATE
	TOKEN_PROTECTED
	TOKEN_INTERNAL
	TOKEN_ABSTRACT
	TOKEN_VIRTUAL
	TOKEN_OVERRIDE
	TOKEN_SEALED
	TOKEN_VOID
	TOKEN_VAR
	TOKEN_REF
	TOKEN_OUT
	TOKEN_PARAMS
	TOKEN_TYPEOF
	TOKEN_IS
	TOKEN_AS
	TOKEN_THROW
	TOKEN_TRY
	TOKEN_CATCH
	TOKEN_FINALLY
	TOKEN_EXTERN

	// Annotations
	TOKEN_AT // @

	// Primitive types
	TOKEN_INT
	TOKEN_UINT
	TOKEN_LONG
	TOKEN_ULONG
	TOKEN_SHORT
	TOKEN_USHORT
	TOKEN_BYTE
	TOKEN_SBYTE
	TOKEN_FLOAT
	TOKEN_DOUBLE
	TOKEN_DECIMAL
	TOKEN_BOOL
	TOKEN_CHAR
	TOKEN_STRING
	TOKEN_OBJECT

	// Operators
	TOKEN_PLUS        // +
	TOKEN_MINUS       // -
	TOKEN_STAR        // *
	TOKEN_SLASH       // /
	TOKEN_PERCENT     // %
	TOKEN_AMP         // &
	TOKEN_PIPE        // |
	TOKEN_CARET       // ^
	TOKEN_TILDE       // ~
	TOKEN_BANG        // !
	TOKEN_LT          // <
	TOKEN_GT          // >
	TOKEN_EQ          // =
	TOKEN_PLUS_EQ     // +=
	TOKEN_MINUS_EQ    // -=
	TOKEN_STAR_EQ     // *=
	TOKEN_SLASH_EQ    // /=
	TOKEN_PERCENT_EQ  // %=
	TOKEN_AMP_EQ      // &=
	TOKEN_PIPE_EQ     // |=
	TOKEN_CARET_EQ    // ^=
	TOKEN_LSHIFT      // <<
	TOKEN_RSHIFT      // >>
	TOKEN_LSHIFT_EQ   // <<=
	TOKEN_RSHIFT_EQ   // >>=
	TOKEN_EQ_EQ       // ==
	TOKEN_BANG_EQ     // !=
	TOKEN_LT_EQ       // <=
	TOKEN_GT_EQ       // >=
	TOKEN_AMP_AMP     // &&
	TOKEN_PIPE_PIPE   // ||
	TOKEN_PLUS_PLUS   // ++
	TOKEN_MINUS_MINUS // --
	TOKEN_ARROW       // ->
	TOKEN_DOT         // .
	TOKEN_QUESTION    // ?
	TOKEN_COLON_COLON // ::
	TOKEN_NULL_COAL   // ??
	TOKEN_NULL_ASGN   // ??=

	// Delimiters
	TOKEN_LPAREN    // (
	TOKEN_RPAREN    // )
	TOKEN_LBRACE    // {
	TOKEN_RBRACE    // }
	TOKEN_LBRACKET  // [
	TOKEN_RBRACKET  // ]
	TOKEN_SEMICOLON // ;
	TOKEN_COLON     // :
	TOKEN_COMMA     // ,

	// Special
	TOKEN_EOF
	TOKEN_INVALID
)

var tokenNames = map[TokenType]string{
	TOKEN_INT_LIT: "int_literal", TOKEN_FLOAT_LIT: "float_literal",
	TOKEN_DOUBLE_LIT: "double_literal", TOKEN_STRING_LIT: "string_literal",
	TOKEN_CHAR_LIT: "char_literal", TOKEN_BOOL_LIT: "bool_literal",
	TOKEN_NULL: "null", TOKEN_IDENT: "identifier",
	TOKEN_NAMESPACE: "namespace", TOKEN_CLASS: "class",
	TOKEN_STRUCT: "struct", TOKEN_INTERFACE: "interface",
	TOKEN_ENUM: "enum", TOKEN_USING: "using",
	TOKEN_IF: "if", TOKEN_ELSE: "else",
	TOKEN_WHILE: "while", TOKEN_FOR: "for",
	TOKEN_FOREACH: "foreach", TOKEN_IN: "in",
	TOKEN_DO: "do", TOKEN_SWITCH: "switch",
	TOKEN_CASE: "case", TOKEN_DEFAULT: "default",
	TOKEN_BREAK: "break", TOKEN_CONTINUE: "continue",
	TOKEN_RETURN: "return", TOKEN_NEW: "new",
	TOKEN_THIS: "this", TOKEN_BASE: "base",
	TOKEN_STATIC: "static", TOKEN_READONLY: "readonly",
	TOKEN_CONST: "const", TOKEN_PUBLIC: "public",
	TOKEN_PRIVATE: "private", TOKEN_PROTECTED: "protected",
	TOKEN_INTERNAL: "internal", TOKEN_ABSTRACT: "abstract",
	TOKEN_VIRTUAL: "virtual", TOKEN_OVERRIDE: "override",
	TOKEN_SEALED: "sealed", TOKEN_VOID: "void",
	TOKEN_VAR: "var", TOKEN_REF: "ref",
	TOKEN_OUT: "out", TOKEN_PARAMS: "params",
	TOKEN_TYPEOF: "typeof", TOKEN_IS: "is",
	TOKEN_AS: "as", TOKEN_THROW: "throw",
	TOKEN_TRY: "try", TOKEN_CATCH: "catch",
	TOKEN_FINALLY: "finally", TOKEN_EXTERN: "extern",
	TOKEN_AT:  "@",
	TOKEN_INT: "int", TOKEN_UINT: "uint",
	TOKEN_LONG: "long", TOKEN_ULONG: "ulong",
	TOKEN_SHORT: "short", TOKEN_USHORT: "ushort",
	TOKEN_BYTE: "byte", TOKEN_SBYTE: "sbyte",
	TOKEN_FLOAT: "float", TOKEN_DOUBLE: "double",
	TOKEN_DECIMAL: "decimal", TOKEN_BOOL: "bool",
	TOKEN_CHAR: "char", TOKEN_STRING: "string",
	TOKEN_OBJECT: "object",
	TOKEN_PLUS:   "+", TOKEN_MINUS: "-",
	TOKEN_STAR: "*", TOKEN_SLASH: "/",
	TOKEN_PERCENT: "%", TOKEN_AMP: "&",
	TOKEN_PIPE: "|", TOKEN_CARET: "^",
	TOKEN_TILDE: "~", TOKEN_BANG: "!",
	TOKEN_LT: "<", TOKEN_GT: ">",
	TOKEN_EQ: "=", TOKEN_PLUS_EQ: "+=",
	TOKEN_MINUS_EQ: "-=", TOKEN_STAR_EQ: "*=",
	TOKEN_SLASH_EQ: "/=", TOKEN_PERCENT_EQ: "%=",
	TOKEN_AMP_EQ: "&=", TOKEN_PIPE_EQ: "|=",
	TOKEN_CARET_EQ: "^=", TOKEN_LSHIFT: "<<",
	TOKEN_RSHIFT: ">>", TOKEN_LSHIFT_EQ: "<<=",
	TOKEN_RSHIFT_EQ: ">>=", TOKEN_EQ_EQ: "==",
	TOKEN_BANG_EQ: "!=", TOKEN_LT_EQ: "<=",
	TOKEN_GT_EQ: ">=", TOKEN_AMP_AMP: "&&",
	TOKEN_PIPE_PIPE: "||", TOKEN_PLUS_PLUS: "++",
	TOKEN_MINUS_MINUS: "--", TOKEN_ARROW: "->",
	TOKEN_DOT: ".", TOKEN_QUESTION: "?",
	TOKEN_COLON_COLON: "::", TOKEN_NULL_COAL: "??",
	TOKEN_NULL_ASGN: "??=",
	TOKEN_LPAREN:    "(", TOKEN_RPAREN: ")",
	TOKEN_LBRACE: "{", TOKEN_RBRACE: "}",
	TOKEN_LBRACKET: "[", TOKEN_RBRACKET: "]",
	TOKEN_SEMICOLON: ";", TOKEN_COLON: ":",
	TOKEN_COMMA: ",",
	TOKEN_EOF:   "EOF", TOKEN_INVALID: "INVALID",
}

func (t TokenType) String() string {
	if s, ok := tokenNames[t]; ok {
		return s
	}
	return fmt.Sprintf("token(%d)", int(t))
}

// Token is a single lexed token with position info
type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Col    int
	File   string
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Lexeme, t.Line, t.Col)
}

// Pos returns a human-readable position string
func (t Token) Pos() string {
	if t.File != "" {
		return fmt.Sprintf("%s:%d:%d", t.File, t.Line, t.Col)
	}
	return fmt.Sprintf("%d:%d", t.Line, t.Col)
}

var keywords = map[string]TokenType{
	"namespace": TOKEN_NAMESPACE,
	"class":     TOKEN_CLASS,
	"struct":    TOKEN_STRUCT,
	"interface": TOKEN_INTERFACE,
	"enum":      TOKEN_ENUM,
	"using":     TOKEN_USING,
	"if":        TOKEN_IF,
	"else":      TOKEN_ELSE,
	"while":     TOKEN_WHILE,
	"for":       TOKEN_FOR,
	"foreach":   TOKEN_FOREACH,
	"in":        TOKEN_IN,
	"do":        TOKEN_DO,
	"switch":    TOKEN_SWITCH,
	"case":      TOKEN_CASE,
	"default":   TOKEN_DEFAULT,
	"break":     TOKEN_BREAK,
	"continue":  TOKEN_CONTINUE,
	"return":    TOKEN_RETURN,
	"new":       TOKEN_NEW,
	"this":      TOKEN_THIS,
	"base":      TOKEN_BASE,
	"static":    TOKEN_STATIC,
	"readonly":  TOKEN_READONLY,
	"const":     TOKEN_CONST,
	"public":    TOKEN_PUBLIC,
	"private":   TOKEN_PRIVATE,
	"protected": TOKEN_PROTECTED,
	"internal":  TOKEN_INTERNAL,
	"abstract":  TOKEN_ABSTRACT,
	"virtual":   TOKEN_VIRTUAL,
	"override":  TOKEN_OVERRIDE,
	"sealed":    TOKEN_SEALED,
	"void":      TOKEN_VOID,
	"var":       TOKEN_VAR,
	"ref":       TOKEN_REF,
	"out":       TOKEN_OUT,
	"params":    TOKEN_PARAMS,
	"typeof":    TOKEN_TYPEOF,
	"is":        TOKEN_IS,
	"as":        TOKEN_AS,
	"throw":     TOKEN_THROW,
	"try":       TOKEN_TRY,
	"catch":     TOKEN_CATCH,
	"finally":   TOKEN_FINALLY,
	"extern":    TOKEN_EXTERN,
	"true":      TOKEN_BOOL_LIT,
	"false":     TOKEN_BOOL_LIT,
	"null":      TOKEN_NULL,
	"int":       TOKEN_INT,
	"uint":      TOKEN_UINT,
	"long":      TOKEN_LONG,
	"ulong":     TOKEN_ULONG,
	"short":     TOKEN_SHORT,
	"ushort":    TOKEN_USHORT,
	"byte":      TOKEN_BYTE,
	"sbyte":     TOKEN_SBYTE,
	"float":     TOKEN_FLOAT,
	"double":    TOKEN_DOUBLE,
	"decimal":   TOKEN_DECIMAL,
	"bool":      TOKEN_BOOL,
	"char":      TOKEN_CHAR,
	"string":    TOKEN_STRING,
	"object":    TOKEN_OBJECT,
}
