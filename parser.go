package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
	file   string
	errors []TranspileError
}

func NewParser(tokens []Token, file string) *Parser {
	return &Parser{tokens: tokens, file: file}
}

func (p *Parser) peek() Token {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type != TOKEN_INVALID {
			return t
		}
		p.pos++
	}
	return Token{Type: TOKEN_EOF}
}

func (p *Parser) peek2() Token {
	old := p.pos
	p.pos++
	t := p.peek()
	p.pos = old
	return t
}

func (p *Parser) advance() Token {
	t := p.peek()
	p.pos++
	return t
}

func (p *Parser) check(tt TokenType) bool { return p.peek().Type == tt }

func (p *Parser) match(types ...TokenType) bool {
	for _, tt := range types {
		if p.peek().Type == tt {
			return true
		}
	}
	return false
}

func (p *Parser) consume(tt TokenType, context string) Token {
	t := p.peek()
	if t.Type != tt {
		p.errorf(t, "P0001", "expected %s %s, found %q", tt, context, t.Lexeme)
		return Token{Type: tt, Lexeme: "", Line: t.Line, Col: t.Col, File: t.File}
	}
	return p.advance()
}

func (p *Parser) errorf(tok Token, code string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors, newError("parser", p.file, tok, code, msg))
}

func (p *Parser) errorHint(tok Token, code string, msg string, hints ...string) {
	p.errors = append(p.errors, newError("parser", p.file, tok, code, msg, hints...))
}

func (p *Parser) sync() {
	for !p.check(TOKEN_EOF) {
		switch p.peek().Type {
		case TOKEN_SEMICOLON:
			p.advance()
			return
		case TOKEN_CLASS, TOKEN_STRUCT, TOKEN_NAMESPACE, TOKEN_RBRACE,
			TOKEN_PUBLIC, TOKEN_PRIVATE, TOKEN_PROTECTED, TOKEN_STATIC,
			TOKEN_IF, TOKEN_FOR, TOKEN_WHILE, TOKEN_RETURN:
			return
		}
		p.advance()
	}
}

// ── Entry point ───────────────────────────────────────────────────────────────

func (p *Parser) ParseFile() (*CompilationUnit, []TranspileError) {
	cu := &CompilationUnit{}
	for p.check(TOKEN_USING) {
		cu.Usings = append(cu.Usings, p.parseUsing())
	}
	for !p.check(TOKEN_EOF) {
		if p.check(TOKEN_NAMESPACE) {
			cu.Namespaces = append(cu.Namespaces, p.parseNamespace())
		} else {
			decl := p.parseTopLevelDecl()
			if decl != nil {
				cu.Members = append(cu.Members, decl)
			}
		}
	}
	return cu, p.errors
}

func (p *Parser) parseUsing() *UsingDecl {
	tok := p.consume(TOKEN_USING, "declaration")
	path := []Token{p.consume(TOKEN_IDENT, "namespace identifier")}
	for p.check(TOKEN_DOT) {
		p.advance()
		path = append(path, p.consume(TOKEN_IDENT, "namespace identifier"))
	}
	p.consume(TOKEN_SEMICOLON, "after using declaration")
	return &UsingDecl{Tok: tok, Path: path}
}

func (p *Parser) parseNamespace() *NamespaceDecl {
	tok := p.consume(TOKEN_NAMESPACE, "declaration")
	path := []Token{p.consume(TOKEN_IDENT, "namespace name")}
	for p.check(TOKEN_DOT) {
		p.advance()
		path = append(path, p.consume(TOKEN_IDENT, "namespace name"))
	}
	p.consume(TOKEN_LBRACE, "opening '{' of namespace")
	var members []Decl
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		if p.check(TOKEN_NAMESPACE) {
			members = append(members, p.parseNamespace())
		} else {
			d := p.parseTopLevelDecl()
			if d != nil {
				members = append(members, d)
			}
		}
	}
	p.consume(TOKEN_RBRACE, "closing '}' of namespace")
	return &NamespaceDecl{Tok: tok, Path: path, Members: members}
}

// ── Modifiers + Annotations ───────────────────────────────────────────────────

type modSet struct {
	access     AccessMod
	isStatic   bool
	isAbstract bool
	isVirtual  bool
	isOverride bool
	isSealed   bool
	isReadonly bool
	isConst    bool
	isExtern   bool
}

func (p *Parser) parseAnnotations() []*Annotation {
	var anns []*Annotation
	for p.check(TOKEN_AT) {
		at := p.advance()
		name := p.consume(TOKEN_IDENT, "annotation name")
		ann := &Annotation{At: at, Name: name}
		if p.check(TOKEN_LPAREN) {
			p.advance()
			for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
				ann.Args = append(ann.Args, p.parseExpr())
				if !p.check(TOKEN_RPAREN) {
					p.consume(TOKEN_COMMA, "between annotation arguments")
				}
			}
			p.consume(TOKEN_RPAREN, "closing ')'")
		}
		anns = append(anns, ann)
	}
	return anns
}

func (p *Parser) parseMods() modSet {
	ms := modSet{}
	for {
		switch p.peek().Type {
		case TOKEN_PUBLIC:
			p.advance()
			ms.access = ACCESS_PUBLIC
		case TOKEN_PRIVATE:
			p.advance()
			ms.access = ACCESS_PRIVATE
		case TOKEN_PROTECTED:
			p.advance()
			ms.access = ACCESS_PROTECTED
		case TOKEN_INTERNAL:
			p.advance()
			ms.access = ACCESS_INTERNAL
		case TOKEN_STATIC:
			p.advance()
			ms.isStatic = true
		case TOKEN_ABSTRACT:
			p.advance()
			ms.isAbstract = true
		case TOKEN_VIRTUAL:
			p.advance()
			ms.isVirtual = true
		case TOKEN_OVERRIDE:
			p.advance()
			ms.isOverride = true
		case TOKEN_SEALED:
			p.advance()
			ms.isSealed = true
		case TOKEN_READONLY:
			p.advance()
			ms.isReadonly = true
		case TOKEN_CONST:
			p.advance()
			ms.isConst = true
		case TOKEN_EXTERN:
			p.advance()
			ms.isExtern = true
		default:
			return ms
		}
	}
}

// ── Top-level declarations ────────────────────────────────────────────────────

func (p *Parser) parseTopLevelDecl() Decl {
	anns := p.parseAnnotations()
	ms := p.parseMods()
	switch p.peek().Type {
	case TOKEN_CLASS:
		return p.parseClass(anns, ms)
	case TOKEN_STRUCT:
		return p.parseStruct(anns, ms)
	case TOKEN_INTERFACE:
		return p.parseInterface(anns, ms)
	case TOKEN_ENUM:
		return p.parseEnum(anns, ms)
	default:
		tok := p.peek()
		p.errorHint(tok, "P0002",
			fmt.Sprintf("unexpected token %q at top level", tok.Lexeme),
			"Top-level declarations must be class, struct, interface, or enum",
			"Did you forget to wrap this in a namespace or class?")
		p.sync()
		return nil
	}
}

func (p *Parser) parseClass(anns []*Annotation, ms modSet) *ClassDecl {
	tok := p.consume(TOKEN_CLASS, "declaration")
	name := p.consume(TOKEN_IDENT, "class name")
	typeParams := p.parseTypeParams()
	baseTypes := p.parseBaseList()
	p.consume(TOKEN_LBRACE, "'{' after class declaration")
	members := p.parseClassBody()
	p.consume(TOKEN_RBRACE, "'}' closing class body")
	return &ClassDecl{
		Annotations: anns, Access: ms.access, IsStatic: ms.isStatic,
		IsAbstract: ms.isAbstract, IsSealed: ms.isSealed,
		Tok: tok, Name: name, TypeParams: typeParams,
		BaseTypes: baseTypes, Members: members,
	}
}

func (p *Parser) parseStruct(anns []*Annotation, ms modSet) *StructDecl {
	tok := p.consume(TOKEN_STRUCT, "declaration")
	name := p.consume(TOKEN_IDENT, "struct name")
	typeParams := p.parseTypeParams()
	baseTypes := p.parseBaseList()
	p.consume(TOKEN_LBRACE, "'{' after struct declaration")
	members := p.parseClassBody()
	p.consume(TOKEN_RBRACE, "'}' closing struct body")
	return &StructDecl{
		Annotations: anns, Access: ms.access, Tok: tok, Name: name,
		TypeParams: typeParams, BaseTypes: baseTypes, Members: members,
	}
}

func (p *Parser) parseInterface(anns []*Annotation, ms modSet) *InterfaceDecl {
	tok := p.consume(TOKEN_INTERFACE, "declaration")
	name := p.consume(TOKEN_IDENT, "interface name")
	typeParams := p.parseTypeParams()
	baseTypes := p.parseBaseList()
	p.consume(TOKEN_LBRACE, "'{' after interface declaration")
	members := p.parseClassBody()
	p.consume(TOKEN_RBRACE, "'}' closing interface body")
	return &InterfaceDecl{
		Annotations: anns, Access: ms.access, Tok: tok, Name: name,
		TypeParams: typeParams, BaseTypes: baseTypes, Members: members,
	}
}

func (p *Parser) parseEnum(anns []*Annotation, ms modSet) *EnumDecl {
	tok := p.consume(TOKEN_ENUM, "declaration")
	name := p.consume(TOKEN_IDENT, "enum name")
	var baseType *TypeNode
	if p.check(TOKEN_COLON) {
		p.advance()
		baseType = p.parseTypeName()
	}
	p.consume(TOKEN_LBRACE, "'{' after enum declaration")
	var members []*EnumMember
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		mAnns := p.parseAnnotations()
		mName := p.consume(TOKEN_IDENT, "enum member name")
		var val Expr
		if p.check(TOKEN_EQ) {
			p.advance()
			val = p.parseExpr()
		}
		members = append(members, &EnumMember{Annotations: mAnns, Name: mName, Value: val})
		if p.check(TOKEN_COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.consume(TOKEN_RBRACE, "'}' closing enum body")
	return &EnumDecl{
		Annotations: anns, Access: ms.access, Tok: tok, Name: name,
		BaseType: baseType, Members: members,
	}
}

func (p *Parser) parseTypeParams() []Token {
	if !p.check(TOKEN_LT) {
		return nil
	}
	p.advance()
	var params []Token
	params = append(params, p.consume(TOKEN_IDENT, "type parameter"))
	for p.check(TOKEN_COMMA) {
		p.advance()
		params = append(params, p.consume(TOKEN_IDENT, "type parameter"))
	}
	p.consume(TOKEN_GT, "closing '>' of type parameter list")
	return params
}

func (p *Parser) parseBaseList() []*TypeNode {
	if !p.check(TOKEN_COLON) {
		return nil
	}
	p.advance()
	var bases []*TypeNode
	bases = append(bases, p.parseTypeName())
	for p.check(TOKEN_COMMA) {
		p.advance()
		bases = append(bases, p.parseTypeName())
	}
	return bases
}

// ── Class body members ────────────────────────────────────────────────────────

func (p *Parser) parseClassBody() []Decl {
	var members []Decl
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		anns := p.parseAnnotations()
		ms := p.parseMods()
		d := p.parseMember(anns, ms)
		if d != nil {
			members = append(members, d)
		}
	}
	return members
}

func (p *Parser) parseMember(anns []*Annotation, ms modSet) Decl {
	tok := p.peek()

	switch tok.Type {
	case TOKEN_CLASS:
		return p.parseClass(anns, ms)
	case TOKEN_STRUCT:
		return p.parseStruct(anns, ms)
	case TOKEN_INTERFACE:
		return p.parseInterface(anns, ms)
	case TOKEN_ENUM:
		return p.parseEnum(anns, ms)
	}

	if tok.Type == TOKEN_TILDE {
		return p.parseDestructor(anns)
	}

	returnType := p.parseTypeName()
	if returnType == nil {
		p.errorHint(p.peek(), "P0003",
			fmt.Sprintf("expected member declaration, got %q", p.peek().Lexeme),
			"Member declarations must start with a type name or modifier keyword")
		p.sync()
		return nil
	}

	name := p.consume(TOKEN_IDENT, "member name")

	if p.check(TOKEN_LPAREN) && returnType.Tok.Type == TOKEN_IDENT &&
		returnType.Tok.Lexeme == name.Lexeme {
		return p.parseConstructor(anns, ms, name)
	}

	if p.check(TOKEN_LPAREN) || p.check(TOKEN_LT) {
		return p.parseMethod(anns, ms, returnType, name)
	}

	if p.check(TOKEN_LBRACE) {
		return p.parseProperty(anns, ms, returnType, name)
	}

	return p.parseField(anns, ms, returnType, name)
}

func (p *Parser) parseConstructor(anns []*Annotation, ms modSet, name Token) *ConstructorDecl {
	params := p.parseParamList()
	var baseArgs []Expr
	if p.check(TOKEN_COLON) {
		p.advance()
		baseTok := p.peek()
		if baseTok.Type != TOKEN_BASE && baseTok.Type != TOKEN_THIS {
			p.errorHint(baseTok, "P0004",
				"expected 'base' or 'this' after ':' in constructor",
				"Use ': base(args...)' to call parent constructor")
		} else {
			p.advance()
		}
		p.consume(TOKEN_LPAREN, "'(' for base/this call")
		baseArgs = p.parseArgList()
		p.consume(TOKEN_RPAREN, "')' after base/this arguments")
	}
	body := p.parseBlock()
	return &ConstructorDecl{
		Annotations: anns, Access: ms.access, Name: name,
		Params: params, BaseArgs: baseArgs, Body: body,
	}
}

func (p *Parser) parseDestructor(anns []*Annotation) *DestructorDecl {
	tok := p.advance()
	name := p.consume(TOKEN_IDENT, "destructor class name")
	p.consume(TOKEN_LPAREN, "'(' after destructor name")
	p.consume(TOKEN_RPAREN, "')' after destructor name")
	body := p.parseBlock()
	return &DestructorDecl{Tok: tok, Name: name, Body: body}
}

func (p *Parser) parseMethod(anns []*Annotation, ms modSet, retType *TypeNode, name Token) *MethodDecl {
	typeParams := p.parseTypeParams()
	params := p.parseParamList()
	var body *Block
	if p.check(TOKEN_LBRACE) {
		body = p.parseBlock()
	} else {
		p.consume(TOKEN_SEMICOLON, "';' after abstract/interface method declaration")
	}
	return &MethodDecl{
		Annotations: anns, Access: ms.access, IsStatic: ms.isStatic,
		IsAbstract: ms.isAbstract, IsVirtual: ms.isVirtual,
		IsOverride: ms.isOverride, IsExtern: ms.isExtern,
		ReturnType: retType, Name: name, TypeParams: typeParams,
		Params: params, Body: body,
	}
}

func (p *Parser) parseProperty(anns []*Annotation, ms modSet, propType *TypeNode, name Token) *PropertyDecl {
	p.consume(TOKEN_LBRACE, "'{' opening property body")
	prop := &PropertyDecl{
		Annotations: anns, Access: ms.access, IsStatic: ms.isStatic,
		IsAbstract: ms.isAbstract, IsVirtual: ms.isVirtual,
		IsOverride: ms.isOverride, Type: propType, Name: name,
	}
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		accMod := ACCESS_DEFAULT
		if p.match(TOKEN_PUBLIC, TOKEN_PRIVATE, TOKEN_PROTECTED) {
			switch p.peek().Type {
			case TOKEN_PUBLIC:
				accMod = ACCESS_PUBLIC
			case TOKEN_PRIVATE:
				accMod = ACCESS_PRIVATE
			case TOKEN_PROTECTED:
				accMod = ACCESS_PROTECTED
			}
			p.advance()
		}
		kw := p.peek()
		if kw.Lexeme == "get" {
			p.advance()
			acc := &PropertyAccessor{Tok: kw, Access: accMod}
			if p.check(TOKEN_LBRACE) {
				acc.Body = p.parseBlock()
			} else {
				p.consume(TOKEN_SEMICOLON, "';' after auto-property getter")
			}
			prop.Getter = acc
		} else if kw.Lexeme == "set" {
			p.advance()
			acc := &PropertyAccessor{Tok: kw, Access: accMod}
			if p.check(TOKEN_LBRACE) {
				acc.Body = p.parseBlock()
			} else {
				p.consume(TOKEN_SEMICOLON, "';' after auto-property setter")
			}
			prop.Setter = acc
		} else {
			p.errorHint(kw, "P0005",
				fmt.Sprintf("expected 'get' or 'set' in property, found %q", kw.Lexeme))
			p.sync()
			break
		}
	}
	p.consume(TOKEN_RBRACE, "'}' closing property body")
	return prop
}

func (p *Parser) parseField(anns []*Annotation, ms modSet, fieldType *TypeNode, name Token) *FieldDecl {
	var init Expr
	if p.check(TOKEN_EQ) {
		p.advance()
		init = p.parseExpr()
	}
	p.consume(TOKEN_SEMICOLON, "';' after field declaration")
	return &FieldDecl{
		Annotations: anns, Access: ms.access, IsStatic: ms.isStatic,
		IsReadonly: ms.isReadonly, IsConst: ms.isConst,
		Type: fieldType, Name: name, Init: init,
	}
}

// ── Type name parsing ─────────────────────────────────────────────────────────

func (p *Parser) parseTypeName() *TypeNode {
	tok := p.peek()
	var base *TypeNode

	switch tok.Type {
	case TOKEN_VOID:
		p.advance()
		base = &TypeNode{Tok: tok}
	case TOKEN_INT, TOKEN_UINT, TOKEN_LONG, TOKEN_ULONG, TOKEN_SHORT, TOKEN_USHORT,
		TOKEN_BYTE, TOKEN_SBYTE, TOKEN_FLOAT, TOKEN_DOUBLE, TOKEN_BOOL,
		TOKEN_CHAR, TOKEN_STRING, TOKEN_OBJECT, TOKEN_DECIMAL:
		p.advance()
		base = &TypeNode{Tok: tok}
	case TOKEN_IDENT:
		p.advance()
		base = &TypeNode{Tok: tok}
		for p.check(TOKEN_DOT) && p.peek2().Type == TOKEN_IDENT {
			dotTok := p.advance()
			nextTok := p.advance()
			combined := Token{
				Type:   TOKEN_IDENT,
				Lexeme: base.Tok.Lexeme + "." + nextTok.Lexeme,
				Line:   dotTok.Line, Col: dotTok.Col, File: dotTok.File,
			}
			base = &TypeNode{Tok: combined}
		}
	default:
		return nil
	}

	if p.check(TOKEN_LT) {
		p.advance()
		base.Generic = append(base.Generic, p.parseTypeName())
		for p.check(TOKEN_COMMA) {
			p.advance()
			base.Generic = append(base.Generic, p.parseTypeName())
		}
		p.consume(TOKEN_GT, "closing '>' of generic type")
	}

	if p.check(TOKEN_QUESTION) {
		p.advance()
		base.IsNullable = true
	}

	if p.check(TOKEN_LBRACKET) {
		p.advance()
		rank := 1
		for p.check(TOKEN_COMMA) {
			p.advance()
			rank++
		}
		p.consume(TOKEN_RBRACKET, "']' closing array type")
		base.IsArray = true
		base.ArrayRank = rank
	}

	return base
}

// ── Parameters ────────────────────────────────────────────────────────────────

func (p *Parser) parseParamList() []*Param {
	p.consume(TOKEN_LPAREN, "'(' before parameter list")
	var params []*Param
	for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
		anns := p.parseAnnotations()
		var mod Token
		if p.match(TOKEN_REF, TOKEN_OUT, TOKEN_PARAMS) {
			mod = p.advance()
		}
		typ := p.parseTypeName()
		if typ == nil {
			p.errorHint(p.peek(), "P0006",
				"expected parameter type",
				"Parameters must have an explicit type: void Foo(int x, string y)")
			p.sync()
			break
		}
		name := p.consume(TOKEN_IDENT, "parameter name")
		var def Expr
		if p.check(TOKEN_EQ) {
			p.advance()
			def = p.parseExpr()
		}
		params = append(params, &Param{
			Annotations: anns, Modifier: mod, Type: typ, Name: name, Default: def,
		})
		if p.check(TOKEN_COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.consume(TOKEN_RPAREN, "')' closing parameter list")
	return params
}

// ── Statements ────────────────────────────────────────────────────────────────

func (p *Parser) parseBlock() *Block {
	lbrace := p.consume(TOKEN_LBRACE, "'{' opening block")
	var stmts []Stmt
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmts = append(stmts, s)
		}
	}
	rbrace := p.consume(TOKEN_RBRACE, "'}' closing block")
	return &Block{LBrace: lbrace, Stmts: stmts, RBrace: rbrace}
}

func (p *Parser) parseStatement() Stmt {
	switch p.peek().Type {
	case TOKEN_LBRACE:
		return p.parseBlock()
	case TOKEN_IF:
		return p.parseIf()
	case TOKEN_WHILE:
		return p.parseWhile()
	case TOKEN_DO:
		return p.parseDoWhile()
	case TOKEN_FOR:
		return p.parseFor()
	case TOKEN_FOREACH:
		return p.parseForeach()
	case TOKEN_SWITCH:
		return p.parseSwitch()
	case TOKEN_RETURN:
		return p.parseReturn()
	case TOKEN_BREAK:
		tok := p.advance()
		p.consume(TOKEN_SEMICOLON, "';' after break")
		return &BreakStmt{Tok: tok}
	case TOKEN_CONTINUE:
		tok := p.advance()
		p.consume(TOKEN_SEMICOLON, "';' after continue")
		return &ContinueStmt{Tok: tok}
	case TOKEN_THROW:
		return p.parseThrow()
	case TOKEN_TRY:
		return p.parseTryCatch()
	case TOKEN_VAR:
		return p.parseVarDecl()
	case TOKEN_CONST:
		p.advance()
		return p.parseTypedDecl(true)
	}

	if p.isTypeStart() && p.isLocalVarDecl() {
		return p.parseTypedDecl(false)
	}

	expr := p.parseExpr()
	semi := p.consume(TOKEN_SEMICOLON, "';' after expression statement")
	return &ExprStmt{Expr: expr, Semi: semi}
}

func (p *Parser) isTypeStart() bool {
	switch p.peek().Type {
	case TOKEN_INT, TOKEN_UINT, TOKEN_LONG, TOKEN_ULONG, TOKEN_SHORT, TOKEN_USHORT,
		TOKEN_BYTE, TOKEN_SBYTE, TOKEN_FLOAT, TOKEN_DOUBLE, TOKEN_BOOL,
		TOKEN_CHAR, TOKEN_STRING, TOKEN_OBJECT, TOKEN_IDENT:
		return true
	}
	return false
}

func (p *Parser) isLocalVarDecl() bool {
	saved := p.pos
	defer func() { p.pos = saved }()
	tn := p.parseTypeName()
	if tn == nil {
		return false
	}
	if p.peek().Type != TOKEN_IDENT {
		return false
	}
	p.advance()
	switch p.peek().Type {
	case TOKEN_EQ, TOKEN_SEMICOLON, TOKEN_COMMA:
		return true
	}
	return false
}

func (p *Parser) parseVarDecl() *LocalVarDecl {
	varTok := p.advance()
	name := p.consume(TOKEN_IDENT, "variable name after 'var'")
	var init Expr
	if p.check(TOKEN_EQ) {
		p.advance()
		init = p.parseExpr()
	} else {
		p.errorHint(name, "P0007",
			"'var' declarations must have an initializer",
			"Change 'var x;' to 'var x = <expression>;'",
			"Or use an explicit type: 'int x;'")
	}
	p.consume(TOKEN_SEMICOLON, "';' after variable declaration")
	return &LocalVarDecl{Type: varTok, TypeNode: nil, Name: name, Init: init}
}

func (p *Parser) parseTypedDecl(isConst bool) Stmt {
	typ := p.parseTypeName()
	if typ == nil {
		tok := p.peek()
		p.errorf(tok, "P0008", "expected type name in local declaration")
		p.sync()
		return nil
	}
	name := p.consume(TOKEN_IDENT, "variable name")
	var init Expr
	if p.check(TOKEN_EQ) {
		p.advance()
		init = p.parseExpr()
	}
	if isConst && init == nil {
		p.errorHint(name, "P0009",
			"const declaration must have an initializer",
			"Add '= <value>' after the variable name")
	}
	p.consume(TOKEN_SEMICOLON, "';' after local variable declaration")
	return &LocalVarDecl{IsConst: isConst, Type: typ.Tok, TypeNode: typ, Name: name, Init: init}
}

func (p *Parser) parseIf() *IfStmt {
	tok := p.advance()
	p.consume(TOKEN_LPAREN, "'(' after if")
	cond := p.parseExpr()
	p.consume(TOKEN_RPAREN, "')' after if condition")
	then := p.parseStatement()
	var elseTok Token
	var elseStmt Stmt
	if p.check(TOKEN_ELSE) {
		elseTok = p.advance()
		elseStmt = p.parseStatement()
	}
	return &IfStmt{Tok: tok, Cond: cond, Then: then, ElseTok: elseTok, Else: elseStmt}
}

func (p *Parser) parseWhile() *WhileStmt {
	tok := p.advance()
	p.consume(TOKEN_LPAREN, "'(' after while")
	cond := p.parseExpr()
	p.consume(TOKEN_RPAREN, "')' after while condition")
	body := p.parseStatement()
	return &WhileStmt{Tok: tok, Cond: cond, Body: body}
}

func (p *Parser) parseDoWhile() *DoWhileStmt {
	tok := p.advance()
	body := p.parseStatement()
	p.consume(TOKEN_WHILE, "'while' after do body")
	p.consume(TOKEN_LPAREN, "'(' after while")
	cond := p.parseExpr()
	p.consume(TOKEN_RPAREN, "')' after do-while condition")
	p.consume(TOKEN_SEMICOLON, "';' after do-while")
	return &DoWhileStmt{Tok: tok, Body: body, Cond: cond}
}

func (p *Parser) parseFor() *ForStmt {
	tok := p.advance()
	p.consume(TOKEN_LPAREN, "'(' after for")

	var init Stmt
	if !p.check(TOKEN_SEMICOLON) {
		if p.isTypeStart() && p.isLocalVarDecl() {
			init = p.parseTypedDecl(false)
		} else {
			expr := p.parseExpr()
			p.consume(TOKEN_SEMICOLON, "';' after for init")
			init = &ExprStmt{Expr: expr}
		}
	} else {
		p.advance()
	}

	var cond Expr
	if !p.check(TOKEN_SEMICOLON) {
		cond = p.parseExpr()
	}
	p.consume(TOKEN_SEMICOLON, "';' after for condition")

	var post []Expr
	for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
		post = append(post, p.parseExpr())
		if p.check(TOKEN_COMMA) {
			p.advance()
		}
	}
	p.consume(TOKEN_RPAREN, "')' after for clauses")
	body := p.parseStatement()
	return &ForStmt{Tok: tok, Init: init, Cond: cond, Post: post, Body: body}
}

func (p *Parser) parseForeach() *ForeachStmt {
	tok := p.advance()
	p.consume(TOKEN_LPAREN, "'(' after foreach")
	elemType := p.parseTypeName()
	if elemType == nil {
		p.errorHint(p.peek(), "P0010",
			"expected element type in foreach",
			"Use: foreach (int item in myArray)")
		p.sync()
		return nil
	}
	elemName := p.consume(TOKEN_IDENT, "element variable name")
	p.consume(TOKEN_IN, "'in' in foreach")
	rangeExpr := p.parseExpr()
	p.consume(TOKEN_RPAREN, "')' after foreach range")
	body := p.parseStatement()
	return &ForeachStmt{
		Tok: tok, ElemType: elemType, ElemName: elemName,
		Range: rangeExpr, Body: body,
	}
}

func (p *Parser) parseSwitch() *SwitchStmt {
	tok := p.advance()
	p.consume(TOKEN_LPAREN, "'(' after switch")
	expr := p.parseExpr()
	p.consume(TOKEN_RPAREN, "')' after switch expression")
	p.consume(TOKEN_LBRACE, "'{' opening switch body")
	var cases []*SwitchCase
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		caseTok := p.peek()
		var val Expr
		if p.check(TOKEN_CASE) {
			p.advance()
			val = p.parseExpr()
			p.consume(TOKEN_COLON, "':' after case value")
		} else if p.check(TOKEN_DEFAULT) {
			p.advance()
			p.consume(TOKEN_COLON, "':' after default")
		} else {
			p.errorf(p.peek(), "P0011", "expected 'case' or 'default' in switch")
			p.sync()
			break
		}
		var body []Stmt
		for !p.check(TOKEN_CASE) && !p.check(TOKEN_DEFAULT) &&
			!p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
			s := p.parseStatement()
			if s != nil {
				body = append(body, s)
			}
		}
		cases = append(cases, &SwitchCase{Tok: caseTok, Value: val, Body: body})
	}
	p.consume(TOKEN_RBRACE, "'}' closing switch body")
	return &SwitchStmt{Tok: tok, Expr: expr, Cases: cases}
}

func (p *Parser) parseReturn() *ReturnStmt {
	tok := p.advance()
	var val Expr
	if !p.check(TOKEN_SEMICOLON) {
		val = p.parseExpr()
	}
	p.consume(TOKEN_SEMICOLON, "';' after return")
	return &ReturnStmt{Tok: tok, Value: val}
}

func (p *Parser) parseThrow() *ThrowStmt {
	tok := p.advance()
	expr := p.parseExpr()
	p.consume(TOKEN_SEMICOLON, "';' after throw")
	return &ThrowStmt{Tok: tok, Expr: expr}
}

func (p *Parser) parseTryCatch() *TryCatchStmt {
	tok := p.advance()
	body := p.parseBlock()
	var catches []*CatchClause
	var finally *Block
	for p.check(TOKEN_CATCH) {
		catchTok := p.advance()
		p.consume(TOKEN_LPAREN, "'(' after catch")
		excType := p.parseTypeName()
		var excName Token
		if p.check(TOKEN_IDENT) {
			excName = p.advance()
		}
		p.consume(TOKEN_RPAREN, "')' after catch type")
		catchBody := p.parseBlock()
		catches = append(catches, &CatchClause{
			Tok: catchTok, ExcType: excType, ExcName: excName, Body: catchBody,
		})
	}
	if p.check(TOKEN_FINALLY) {
		p.advance()
		finally = p.parseBlock()
	}
	if len(catches) == 0 && finally == nil {
		p.errorHint(tok, "P0012",
			"try without catch or finally is meaningless",
			"Add a catch clause: catch (Exception e) { ... }",
			"Or add a finally block: finally { ... }")
	}
	return &TryCatchStmt{Tok: tok, Body: body, Catches: catches, Finally: finally}
}

// ── Expressions ───────────────────────────────────────────────────────────────

func (p *Parser) parseArgList() []Expr {
	var args []Expr
	for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
		args = append(args, p.parseExpr())
		if p.check(TOKEN_COMMA) {
			p.advance()
		} else {
			break
		}
	}
	return args
}

func (p *Parser) parseExpr() Expr { return p.parseAssign() }

func (p *Parser) parseAssign() Expr {
	left := p.parseTernary()
	switch p.peek().Type {
	case TOKEN_EQ, TOKEN_PLUS_EQ, TOKEN_MINUS_EQ, TOKEN_STAR_EQ, TOKEN_SLASH_EQ,
		TOKEN_PERCENT_EQ, TOKEN_AMP_EQ, TOKEN_PIPE_EQ, TOKEN_CARET_EQ,
		TOKEN_LSHIFT_EQ, TOKEN_RSHIFT_EQ, TOKEN_NULL_ASGN:
		op := p.advance()
		right := p.parseAssign()
		return &AssignExpr{Target: left, Op: op, Value: right}
	}
	return left
}

func (p *Parser) parseTernary() Expr {
	cond := p.parseNullCoal()
	if p.check(TOKEN_QUESTION) {
		qm := p.advance()
		then := p.parseExpr()
		p.consume(TOKEN_COLON, "':' in ternary expression")
		els := p.parseExpr()
		return &TernaryExpr{Cond: cond, Then: then, Else: els, QMark: qm}
	}
	return cond
}

func (p *Parser) parseNullCoal() Expr {
	left := p.parseOr()
	for p.check(TOKEN_NULL_COAL) {
		op := p.advance()
		right := p.parseOr()
		left = &NullCoalExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseOr() Expr {
	left := p.parseAnd()
	for p.check(TOKEN_PIPE_PIPE) {
		op := p.advance()
		right := p.parseAnd()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseAnd() Expr {
	left := p.parseBitOr()
	for p.check(TOKEN_AMP_AMP) {
		op := p.advance()
		right := p.parseBitOr()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseBitOr() Expr {
	left := p.parseBitXor()
	for p.check(TOKEN_PIPE) {
		op := p.advance()
		right := p.parseBitXor()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseBitXor() Expr {
	left := p.parseBitAnd()
	for p.check(TOKEN_CARET) {
		op := p.advance()
		right := p.parseBitAnd()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseBitAnd() Expr {
	left := p.parseEquality()
	for p.check(TOKEN_AMP) {
		op := p.advance()
		right := p.parseEquality()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseEquality() Expr {
	left := p.parseRelational()
	for p.check(TOKEN_EQ_EQ) || p.check(TOKEN_BANG_EQ) {
		op := p.advance()
		right := p.parseRelational()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseRelational() Expr {
	left := p.parseShift()
	for {
		switch p.peek().Type {
		case TOKEN_LT, TOKEN_GT, TOKEN_LT_EQ, TOKEN_GT_EQ:
			op := p.advance()
			right := p.parseShift()
			left = &BinaryExpr{Left: left, Op: op, Right: right}
			continue
		case TOKEN_IS:
			tok := p.advance()
			typ := p.parseTypeName()
			left = &IsExpr{Expr: left, Tok: tok, Type: typ}
			continue
		case TOKEN_AS:
			tok := p.advance()
			typ := p.parseTypeName()
			left = &AsExpr{Expr: left, Tok: tok, Type: typ}
			continue
		}
		break
	}
	return left
}

func (p *Parser) parseShift() Expr {
	left := p.parseAddSub()
	for p.check(TOKEN_LSHIFT) || p.check(TOKEN_RSHIFT) {
		op := p.advance()
		right := p.parseAddSub()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseAddSub() Expr {
	left := p.parseMulDiv()
	for p.check(TOKEN_PLUS) || p.check(TOKEN_MINUS) {
		op := p.advance()
		right := p.parseMulDiv()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseMulDiv() Expr {
	left := p.parseUnary()
	for p.check(TOKEN_STAR) || p.check(TOKEN_SLASH) || p.check(TOKEN_PERCENT) {
		op := p.advance()
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseUnary() Expr {
	switch p.peek().Type {
	case TOKEN_BANG, TOKEN_MINUS, TOKEN_PLUS, TOKEN_TILDE:
		op := p.advance()
		operand := p.parseUnary()
		return &UnaryExpr{Op: op, Operand: operand}
	case TOKEN_PLUS_PLUS, TOKEN_MINUS_MINUS:
		op := p.advance()
		operand := p.parseUnary()
		return &UnaryExpr{Op: op, Operand: operand}
	case TOKEN_LPAREN:
		if cast := p.tryCast(); cast != nil {
			return cast
		}
	}
	return p.parsePostfix()
}

func (p *Parser) tryCast() *CastExpr {
	saved := p.pos
	lp := p.advance()
	typ := p.parseTypeName()
	if typ == nil || !p.check(TOKEN_RPAREN) {
		p.pos = saved
		return nil
	}
	p.advance()
	next := p.peek()
	switch next.Type {
	case TOKEN_INT_LIT, TOKEN_FLOAT_LIT, TOKEN_DOUBLE_LIT, TOKEN_STRING_LIT,
		TOKEN_CHAR_LIT, TOKEN_BOOL_LIT, TOKEN_NULL, TOKEN_IDENT,
		TOKEN_LPAREN, TOKEN_NEW, TOKEN_THIS, TOKEN_BANG, TOKEN_MINUS,
		TOKEN_PLUS, TOKEN_TILDE, TOKEN_PLUS_PLUS, TOKEN_MINUS_MINUS:
		expr := p.parseUnary()
		return &CastExpr{LParen: lp, Type: typ, Expr: expr}
	}
	p.pos = saved
	return nil
}

func (p *Parser) parsePostfix() Expr {
	expr := p.parsePrimary()
	for {
		switch p.peek().Type {
		case TOKEN_DOT:
			dot := p.advance()
			member := p.consume(TOKEN_IDENT, "member name after '.'")
			expr = &MemberExpr{Object: expr, Dot: dot, Member: member}
		case TOKEN_LBRACKET:
			lb := p.advance()
			idx := p.parseExpr()
			p.consume(TOKEN_RBRACKET, "']' after index")
			expr = &IndexExpr{Object: expr, LBracket: lb, Index: idx}
		case TOKEN_LPAREN:
			lp := p.advance()
			args := p.parseArgList()
			p.consume(TOKEN_RPAREN, "')' after call arguments")
			expr = &CallExpr{Callee: expr, LParen: lp, Args: args}
		case TOKEN_PLUS_PLUS, TOKEN_MINUS_MINUS:
			op := p.advance()
			expr = &UnaryExpr{Op: op, Operand: expr, IsPost: true}
		default:
			return expr
		}
	}
}

func (p *Parser) parsePrimary() Expr {
	tok := p.peek()
	switch tok.Type {
	case TOKEN_INT_LIT:
		p.advance()
		raw := strings.ReplaceAll(tok.Lexeme, "_", "")
		raw = strings.TrimRight(raw, "LlUu")
		var v int64
		if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
			v, _ = strconv.ParseInt(raw[2:], 16, 64)
		} else {
			v, _ = strconv.ParseInt(raw, 10, 64)
		}
		return &IntLit{Tok: tok, Val: v}
	case TOKEN_FLOAT_LIT:
		p.advance()
		raw := strings.TrimRight(tok.Lexeme, "fF")
		raw = strings.ReplaceAll(raw, "_", "")
		v, _ := strconv.ParseFloat(raw, 32)
		return &FloatLit{Tok: tok, Val: v}
	case TOKEN_DOUBLE_LIT:
		p.advance()
		raw := strings.TrimRight(tok.Lexeme, "dD")
		raw = strings.ReplaceAll(raw, "_", "")
		v, _ := strconv.ParseFloat(raw, 64)
		return &DoubleLit{Tok: tok, Val: v}
	case TOKEN_STRING_LIT:
		p.advance()
		return &StringLit{Tok: tok, Val: tok.Lexeme}
	case TOKEN_CHAR_LIT:
		p.advance()
		var r rune
		if len(tok.Lexeme) > 0 {
			r = rune(tok.Lexeme[0])
		}
		return &CharLit{Tok: tok, Val: r}
	case TOKEN_BOOL_LIT:
		p.advance()
		return &BoolLit{Tok: tok, Val: tok.Lexeme == "true"}
	case TOKEN_NULL:
		p.advance()
		return &NullLit{Tok: tok}
	case TOKEN_THIS:
		p.advance()
		return &ThisExpr{Tok: tok}
	case TOKEN_BASE:
		p.advance()
		return &BaseExpr{Tok: tok}
	case TOKEN_TYPEOF:
		p.advance()
		p.consume(TOKEN_LPAREN, "'(' after typeof")
		typ := p.parseTypeName()
		p.consume(TOKEN_RPAREN, "')' closing typeof")
		return &TypeofExpr{Tok: tok, Type: typ}
	case TOKEN_NEW:
		return p.parseNew()
	case TOKEN_LPAREN:
		p.advance()
		e := p.parseExpr()
		p.consume(TOKEN_RPAREN, "')' closing grouped expression")
		return e
	case TOKEN_LBRACE:
		return p.parseArrayInit()
	case TOKEN_IDENT:
		p.advance()
		return &IdentExpr{Tok: tok}
	case TOKEN_INT, TOKEN_LONG, TOKEN_FLOAT, TOKEN_DOUBLE, TOKEN_BOOL,
		TOKEN_STRING, TOKEN_CHAR, TOKEN_BYTE, TOKEN_UINT, TOKEN_ULONG,
		TOKEN_SHORT, TOKEN_USHORT, TOKEN_SBYTE, TOKEN_OBJECT:
		p.advance()
		return &IdentExpr{Tok: tok}
	}
	p.errorHint(tok, "P0013",
		fmt.Sprintf("unexpected token %q in expression", tok.Lexeme),
		"Expressions must start with a value, identifier, or operator")
	p.advance()
	return &IntLit{Tok: tok, Val: 0}
}

func (p *Parser) parseNew() Expr {
	tok := p.advance()
	typ := p.parseTypeName()
	if typ == nil {
		p.errorHint(p.peek(), "P0014",
			"expected type name after 'new'",
			"Use: new ClassName(args) or new int[size]")
		return &NullLit{Tok: tok}
	}

	if p.check(TOKEN_LBRACKET) {
		p.advance()
		var size Expr
		if !p.check(TOKEN_RBRACKET) {
			size = p.parseExpr()
		}
		p.consume(TOKEN_RBRACKET, "']' in array creation")
		var init []Expr
		if p.check(TOKEN_LBRACE) {
			p.advance()
			for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
				init = append(init, p.parseExpr())
				if p.check(TOKEN_COMMA) {
					p.advance()
				}
			}
			p.consume(TOKEN_RBRACE, "'}' closing array initializer")
		}
		return &NewArrayExpr{Tok: tok, ElemType: typ, Size: size, Init: init}
	}

	p.consume(TOKEN_LPAREN, "'(' after new type")
	args := p.parseArgList()
	p.consume(TOKEN_RPAREN, "')' after new arguments")
	return &NewExpr{Tok: tok, Type: typ, Args: args}
}

func (p *Parser) parseArrayInit() *ArrayInitExpr {
	lb := p.advance()
	var elems []Expr
	for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
		elems = append(elems, p.parseExpr())
		if p.check(TOKEN_COMMA) {
			p.advance()
		}
	}
	p.consume(TOKEN_RBRACE, "'}' closing array initializer")
	return &ArrayInitExpr{LBrace: lb, Elems: elems}
}
