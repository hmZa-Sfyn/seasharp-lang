package main

import (
	"fmt"
	"strings"
)

// ── Type system ───────────────────────────────────────────────────────────────

type CsTypeKind int

const (
	TK_VOID CsTypeKind = iota
	TK_INT
	TK_UINT
	TK_LONG
	TK_ULONG
	TK_SHORT
	TK_USHORT
	TK_BYTE
	TK_SBYTE
	TK_FLOAT
	TK_DOUBLE
	TK_BOOL
	TK_CHAR
	TK_STRING
	TK_OBJECT
	TK_CLASS
	TK_STRUCT
	TK_INTERFACE
	TK_ENUM
	TK_ARRAY
	TK_GENERIC
	TK_NULL
	TK_UNKNOWN
)

type CsType struct {
	Kind       CsTypeKind
	Name       string // fully qualified name
	IsNullable bool
	ElemType   *CsType   // for arrays
	TypeArgs   []*CsType // for generics
	Decl       Decl      // backing declaration (ClassDecl etc.)
}

func (t *CsType) String() string {
	if t == nil {
		return "<nil>"
	}
	s := t.Name
	if len(t.TypeArgs) > 0 {
		args := make([]string, len(t.TypeArgs))
		for i, a := range t.TypeArgs {
			args[i] = a.String()
		}
		s += "<" + strings.Join(args, ", ") + ">"
	}
	if t.Kind == TK_ARRAY {
		s += "[]"
	}
	if t.IsNullable {
		s += "?"
	}
	return s
}

func (t *CsType) IsNumeric() bool {
	switch t.Kind {
	case TK_INT, TK_UINT, TK_LONG, TK_ULONG, TK_SHORT, TK_USHORT,
		TK_BYTE, TK_SBYTE, TK_FLOAT, TK_DOUBLE:
		return true
	}
	return false
}

func (t *CsType) IsIntegral() bool {
	switch t.Kind {
	case TK_INT, TK_UINT, TK_LONG, TK_ULONG, TK_SHORT, TK_USHORT, TK_BYTE, TK_SBYTE:
		return true
	}
	return false
}

// ── Symbol table / scope ──────────────────────────────────────────────────────

type Symbol struct {
	Name     string
	Type     *CsType
	Tok      Token
	IsConst  bool
	IsStatic bool
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

func newScope(parent *Scope) *Scope {
	return &Scope{parent: parent, symbols: make(map[string]*Symbol)}
}

func (s *Scope) define(sym *Symbol) bool {
	if _, exists := s.symbols[sym.Name]; exists {
		return false
	}
	s.symbols[sym.Name] = sym
	return true
}

func (s *Scope) lookup(name string) *Symbol {
	if sym, ok := s.symbols[name]; ok {
		return sym
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return nil
}

// ── Type checker ──────────────────────────────────────────────────────────────

type TypeChecker struct {
	file       string
	errors     []TranspileError
	scope      *Scope
	typeMap    map[string]*CsType // all declared types
	currentRet *CsType            // return type of current method
	inLoop     int                // nesting level for break/continue
	inSwitch   int
}

func NewTypeChecker(file string) *TypeChecker {
	tc := &TypeChecker{
		file:    file,
		typeMap: make(map[string]*CsType),
		scope:   newScope(nil),
	}
	tc.registerBuiltins()
	return tc
}

func (tc *TypeChecker) registerBuiltins() {
	primitives := []struct {
		name string
		kind CsTypeKind
	}{
		{"void", TK_VOID}, {"int", TK_INT}, {"uint", TK_UINT},
		{"long", TK_LONG}, {"ulong", TK_ULONG}, {"short", TK_SHORT},
		{"ushort", TK_USHORT}, {"byte", TK_BYTE}, {"sbyte", TK_SBYTE},
		{"float", TK_FLOAT}, {"double", TK_DOUBLE}, {"bool", TK_BOOL},
		{"char", TK_CHAR}, {"string", TK_STRING}, {"object", TK_OBJECT},
	}
	for _, p := range primitives {
		tc.typeMap[p.name] = &CsType{Kind: p.kind, Name: p.name}
	}
	// Register common exception types
	for _, ex := range []string{"Exception", "InvalidOperationException",
		"ArgumentException", "ArgumentNullException", "NullReferenceException",
		"IndexOutOfRangeException", "NotImplementedException"} {
		tc.typeMap[ex] = &CsType{Kind: TK_CLASS, Name: ex}
	}
}

func (tc *TypeChecker) errorf(tok Token, code, format string, args ...interface{}) {
	tc.errors = append(tc.errors, newError("typechecker", tc.file, tok, code,
		fmt.Sprintf(format, args...)))
}

func (tc *TypeChecker) warnf(tok Token, code, format string, args ...interface{}) {
	tc.errors = append(tc.errors, newWarning("typechecker", tc.file, tok, code,
		fmt.Sprintf(format, args...)))
}

func (tc *TypeChecker) hint(tok Token, code, msg string, hints ...string) {
	tc.errors = append(tc.errors, newError("typechecker", tc.file, tok, code, msg, hints...))
}

func (tc *TypeChecker) Check(cu *CompilationUnit) []TranspileError {
	// First pass: register all type names
	for _, ns := range cu.Namespaces {
		tc.registerNsTypes(ns, "")
	}
	for _, d := range cu.Members {
		tc.registerDecl(d, "")
	}

	// Second pass: full check
	for _, ns := range cu.Namespaces {
		tc.checkNamespace(ns)
	}
	for _, d := range cu.Members {
		tc.checkDecl(d)
	}
	return tc.errors
}

func (tc *TypeChecker) registerNsTypes(ns *NamespaceDecl, prefix string) {
	nsName := nsPath(ns.Path)
	if prefix != "" {
		nsName = prefix + "." + nsName
	}
	for _, d := range ns.Members {
		tc.registerDecl(d, nsName)
	}
}

func nsPath(path []Token) string {
	parts := make([]string, len(path))
	for i, t := range path {
		parts[i] = t.Lexeme
	}
	return strings.Join(parts, ".")
}

func (tc *TypeChecker) registerDecl(d Decl, ns string) {
	switch v := d.(type) {
	case *ClassDecl:
		name := qualName(ns, v.Name.Lexeme)
		tc.typeMap[name] = &CsType{Kind: TK_CLASS, Name: name, Decl: d}
		tc.typeMap[v.Name.Lexeme] = tc.typeMap[name] // short name too
	case *StructDecl:
		name := qualName(ns, v.Name.Lexeme)
		tc.typeMap[name] = &CsType{Kind: TK_STRUCT, Name: name, Decl: d}
		tc.typeMap[v.Name.Lexeme] = tc.typeMap[name]
	case *InterfaceDecl:
		name := qualName(ns, v.Name.Lexeme)
		tc.typeMap[name] = &CsType{Kind: TK_INTERFACE, Name: name, Decl: d}
		tc.typeMap[v.Name.Lexeme] = tc.typeMap[name]
	case *EnumDecl:
		name := qualName(ns, v.Name.Lexeme)
		tc.typeMap[name] = &CsType{Kind: TK_ENUM, Name: name, Decl: d}
		tc.typeMap[v.Name.Lexeme] = tc.typeMap[name]
	case *NamespaceDecl:
		tc.registerNsTypes(v, ns)
	}
}

func qualName(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "." + name
}

func (tc *TypeChecker) checkNamespace(ns *NamespaceDecl) {
	for _, d := range ns.Members {
		if nested, ok := d.(*NamespaceDecl); ok {
			tc.checkNamespace(nested)
		} else {
			tc.checkDecl(d)
		}
	}
}

func (tc *TypeChecker) checkDecl(d Decl) {
	switch v := d.(type) {
	case *ClassDecl:
		tc.checkClass(v)
	case *StructDecl:
		tc.checkStruct(v)
	case *InterfaceDecl:
		tc.checkInterface(v)
	case *EnumDecl:
		tc.checkEnum(v)
	}
}

// ── Class / Struct checks ─────────────────────────────────────────────────────

func (tc *TypeChecker) checkClass(c *ClassDecl) {
	outer := tc.scope
	tc.scope = newScope(outer)

	// Register 'this'
	selfType := tc.typeMap[c.Name.Lexeme]
	if selfType == nil {
		selfType = &CsType{Kind: TK_CLASS, Name: c.Name.Lexeme}
	}
	tc.scope.define(&Symbol{Name: "this", Type: selfType, Tok: c.Name})

	// Check for @test annotations on non-void methods - warn
	for _, m := range c.Members {
		if md, ok := m.(*MethodDecl); ok {
			tc.checkAnnotations(md.Annotations, md.Name)
		}
	}

	// Check each member
	for _, m := range c.Members {
		tc.checkMember(m, selfType, c.IsAbstract)
	}

	tc.scope = outer
}

func (tc *TypeChecker) checkStruct(s *StructDecl) {
	outer := tc.scope
	tc.scope = newScope(outer)
	selfType := tc.typeMap[s.Name.Lexeme]
	if selfType == nil {
		selfType = &CsType{Kind: TK_STRUCT, Name: s.Name.Lexeme}
	}
	tc.scope.define(&Symbol{Name: "this", Type: selfType, Tok: s.Name})
	for _, m := range s.Members {
		tc.checkMember(m, selfType, false)
	}
	tc.scope = outer
}

func (tc *TypeChecker) checkInterface(i *InterfaceDecl) {
	for _, m := range i.Members {
		if md, ok := m.(*MethodDecl); ok {
			if md.Body != nil {
				tc.errorf(md.Name, "TC0020",
					"interface method '%s' must not have a body", md.Name.Lexeme)
			}
		}
	}
}

func (tc *TypeChecker) checkEnum(e *EnumDecl) {
	seen := make(map[string]bool)
	for _, m := range e.Members {
		if seen[m.Name.Lexeme] {
			tc.errorf(m.Name, "TC0021",
				"duplicate enum member '%s'", m.Name.Lexeme)
		}
		seen[m.Name.Lexeme] = true
	}
}

func (tc *TypeChecker) checkAnnotations(anns []*Annotation, tok Token) {
	for _, ann := range anns {
		switch ann.Name.Lexeme {
		case "test":
			// valid — used for test runner
		case "deprecated":
			// valid — emits warning
		case "inline":
			// valid — hint to gcc
		default:
			tc.warnf(ann.Name, "TC0100",
				"unknown annotation '@%s'", ann.Name.Lexeme)
		}
	}
}

func (tc *TypeChecker) checkMember(d Decl, selfType *CsType, classIsAbstract bool) {
	switch v := d.(type) {
	case *FieldDecl:
		tc.checkField(v)
	case *PropertyDecl:
		tc.checkProperty(v)
	case *MethodDecl:
		tc.checkMethod(v, classIsAbstract)
	case *ConstructorDecl:
		tc.checkConstructor(v, selfType)
	case *DestructorDecl:
		tc.checkDestructor(v)
	case *ClassDecl:
		tc.checkClass(v)
	case *StructDecl:
		tc.checkStruct(v)
	case *EnumDecl:
		tc.checkEnum(v)
	}
}

func (tc *TypeChecker) checkField(f *FieldDecl) {
	ftype := tc.resolveTypeNode(f.Type)
	if ftype == nil {
		tc.errorf(f.Name, "TC0001",
			"unknown type '%s' for field '%s'", f.Type.Tok.Lexeme, f.Name.Lexeme)
		return
	}
	if f.IsConst && f.Init == nil {
		tc.hint(f.Name, "TC0002",
			fmt.Sprintf("const field '%s' must have an initializer", f.Name.Lexeme),
			"Add '= <value>' after the field name")
	}
	if f.Init != nil {
		initType := tc.checkExpr(f.Init)
		tc.expectAssignable(ftype, initType, f.Name, "field initializer")
	}
	// register in scope
	tc.scope.define(&Symbol{Name: f.Name.Lexeme, Type: ftype, Tok: f.Name,
		IsConst: f.IsConst, IsStatic: f.IsStatic})
}

func (tc *TypeChecker) checkProperty(p *PropertyDecl) {
	ptype := tc.resolveTypeNode(p.Type)
	if ptype == nil {
		tc.errorf(p.Name, "TC0003",
			"unknown type '%s' for property '%s'", p.Type.Tok.Lexeme, p.Name.Lexeme)
	}
	if p.Getter != nil && p.Getter.Body != nil {
		tc.checkBlockReturn(p.Getter.Body, ptype, p.Name)
	}
	if p.Setter != nil && p.Setter.Body != nil {
		// setter has implicit 'value' parameter
		outer := tc.scope
		tc.scope = newScope(outer)
		tc.scope.define(&Symbol{Name: "value", Type: ptype, Tok: p.Name})
		tc.checkBlock(p.Setter.Body)
		tc.scope = outer
	}
}

func (tc *TypeChecker) checkMethod(m *MethodDecl, classIsAbstract bool) {
	tc.checkAnnotations(m.Annotations, m.Name)
	retType := tc.resolveTypeNode(m.ReturnType)
	if retType == nil {
		tc.errorf(m.Name, "TC0004",
			"unknown return type '%s' for method '%s'",
			m.ReturnType.Tok.Lexeme, m.Name.Lexeme)
		retType = &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}

	if m.IsAbstract && m.Body != nil {
		tc.errorf(m.Name, "TC0005",
			"abstract method '%s' must not have a body", m.Name.Lexeme,
			"Remove the method body or remove the 'abstract' modifier")
	}
	if !m.IsAbstract && !m.IsExtern && !classIsAbstract && m.Body == nil {
		tc.hint(m.Name, "TC0006",
			fmt.Sprintf("non-abstract method '%s' must have a body", m.Name.Lexeme),
			"Add a method body: { ... }",
			"Or mark the method as 'abstract' or 'extern'")
	}

	outer := tc.scope
	tc.scope = newScope(outer)
	for _, param := range m.Params {
		ptype := tc.resolveTypeNode(param.Type)
		if ptype == nil {
			tc.errorf(param.Name, "TC0007",
				"unknown type for parameter '%s'", param.Name.Lexeme)
			ptype = &CsType{Kind: TK_UNKNOWN, Name: "?"}
		}
		if !tc.scope.define(&Symbol{Name: param.Name.Lexeme, Type: ptype, Tok: param.Name}) {
			tc.errorf(param.Name, "TC0008",
				"duplicate parameter name '%s'", param.Name.Lexeme)
		}
	}

	oldRet := tc.currentRet
	tc.currentRet = retType
	if m.Body != nil {
		if retType.Kind != TK_VOID {
			tc.checkBlockReturn(m.Body, retType, m.Name)
		} else {
			tc.checkBlock(m.Body)
		}
	}
	tc.currentRet = oldRet
	tc.scope = outer

	// Warn if @test annotation but return type is not void
	for _, ann := range m.Annotations {
		if ann.Name.Lexeme == "test" && retType.Kind != TK_VOID {
			tc.warnf(m.Name, "TC0101",
				"@test method '%s' should return void", m.Name.Lexeme,
				"Change return type to void for test methods")
		}
		if ann.Name.Lexeme == "test" && len(m.Params) > 0 {
			tc.warnf(m.Name, "TC0102",
				"@test method '%s' should have no parameters", m.Name.Lexeme)
		}
	}
}

func (tc *TypeChecker) checkConstructor(c *ConstructorDecl, selfType *CsType) {
	outer := tc.scope
	tc.scope = newScope(outer)
	for _, param := range c.Params {
		ptype := tc.resolveTypeNode(param.Type)
		if ptype == nil {
			tc.errorf(param.Name, "TC0007",
				"unknown type for parameter '%s'", param.Name.Lexeme)
			ptype = &CsType{Kind: TK_UNKNOWN, Name: "?"}
		}
		tc.scope.define(&Symbol{Name: param.Name.Lexeme, Type: ptype, Tok: param.Name})
	}
	oldRet := tc.currentRet
	tc.currentRet = &CsType{Kind: TK_VOID, Name: "void"}
	tc.checkBlock(c.Body)
	tc.currentRet = oldRet
	tc.scope = outer
}

func (tc *TypeChecker) checkDestructor(d *DestructorDecl) {
	outer := tc.scope
	tc.scope = newScope(outer)
	oldRet := tc.currentRet
	tc.currentRet = &CsType{Kind: TK_VOID, Name: "void"}
	tc.checkBlock(d.Body)
	tc.currentRet = oldRet
	tc.scope = outer
}

// ── Block / Statement checks ──────────────────────────────────────────────────

func (tc *TypeChecker) checkBlock(b *Block) {
	outer := tc.scope
	tc.scope = newScope(outer)
	for _, s := range b.Stmts {
		tc.checkStmt(s)
	}
	tc.scope = outer
}

// checkBlockReturn ensures every code path in a non-void block returns
func (tc *TypeChecker) checkBlockReturn(b *Block, retType *CsType, nameTok Token) {
	outer := tc.scope
	tc.scope = newScope(outer)
	for _, s := range b.Stmts {
		tc.checkStmt(s)
	}
	tc.scope = outer
	if !tc.blockAlwaysReturns(b) {
		tc.hint(nameTok, "TC0009",
			fmt.Sprintf("not all code paths in '%s' return a value", nameTok.Lexeme),
			"Add a return statement at the end of the method",
			fmt.Sprintf("Expected return type: %s", retType.String()))
	}
}

func (tc *TypeChecker) blockAlwaysReturns(b *Block) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if tc.stmtAlwaysReturns(s) {
			return true
		}
	}
	return false
}

func (tc *TypeChecker) stmtAlwaysReturns(s Stmt) bool {
	switch v := s.(type) {
	case *ReturnStmt:
		return true
	case *ThrowStmt:
		return true
	case *Block:
		return tc.blockAlwaysReturns(v)
	case *IfStmt:
		return v.Else != nil && tc.stmtAlwaysReturns(v.Then) && tc.stmtAlwaysReturns(v.Else)
	case *SwitchStmt:
		hasDefault := false
		for _, c := range v.Cases {
			if c.Value == nil {
				hasDefault = true
			}
			allReturn := false
			for _, cs := range c.Body {
				if tc.stmtAlwaysReturns(cs) {
					allReturn = true
					break
				}
			}
			if !allReturn {
				return false
			}
		}
		return hasDefault
	case *TryCatchStmt:
		bodyRet := tc.blockAlwaysReturns(v.Body)
		if v.Finally != nil && tc.blockAlwaysReturns(v.Finally) {
			return true
		}
		allCatchRet := len(v.Catches) > 0
		for _, c := range v.Catches {
			if !tc.blockAlwaysReturns(c.Body) {
				allCatchRet = false
			}
		}
		return bodyRet && allCatchRet
	case *WhileStmt:
		if lit, ok := v.Cond.(*BoolLit); ok && lit.Val {
			return true // while(true) always runs
		}
		return false
	}
	return false
}

func (tc *TypeChecker) checkStmt(s Stmt) {
	switch v := s.(type) {
	case *Block:
		tc.checkBlock(v)
	case *ExprStmt:
		tc.checkExpr(v.Expr)
		// Warn on useless expressions
		switch v.Expr.(type) {
		case *IntLit, *FloatLit, *DoubleLit, *StringLit, *BoolLit, *NullLit:
			tc.warnf(v.Expr.nodePos(), "TC0200",
				"expression statement has no effect",
				"Did you mean to use this value?")
		}
	case *ReturnStmt:
		tc.checkReturn(v)
	case *LocalVarDecl:
		tc.checkLocalVar(v)
	case *IfStmt:
		tc.checkIf(v)
	case *WhileStmt:
		tc.checkWhile(v)
	case *DoWhileStmt:
		tc.checkDoWhile(v)
	case *ForStmt:
		tc.checkFor(v)
	case *ForeachStmt:
		tc.checkForeach(v)
	case *SwitchStmt:
		tc.checkSwitch(v)
	case *ThrowStmt:
		tc.checkThrow(v)
	case *TryCatchStmt:
		tc.checkTryCatch(v)
	case *BreakStmt:
		if tc.inLoop == 0 && tc.inSwitch == 0 {
			tc.errorf(v.Tok, "TC0010",
				"'break' used outside of loop or switch",
				"'break' can only be used inside for, while, do-while, or switch")
		}
	case *ContinueStmt:
		if tc.inLoop == 0 {
			tc.errorf(v.Tok, "TC0011",
				"'continue' used outside of loop",
				"'continue' can only be used inside for, while, or do-while")
		}
	}
}

func (tc *TypeChecker) checkReturn(r *ReturnStmt) {
	if tc.currentRet == nil {
		return
	}
	if tc.currentRet.Kind == TK_VOID {
		if r.Value != nil {
			tc.errorf(r.Tok, "TC0012",
				"cannot return a value from a void method",
				"Remove the return value or change the method's return type")
		}
	} else {
		if r.Value == nil {
			tc.hint(r.Tok, "TC0013",
				"return statement missing value",
				fmt.Sprintf("Expected a value of type '%s'", tc.currentRet.String()))
		} else {
			retType := tc.checkExpr(r.Value)
			tc.expectAssignable(tc.currentRet, retType, r.Tok, "return value")
		}
	}
}

func (tc *TypeChecker) checkLocalVar(v *LocalVarDecl) {
	var varType *CsType
	if v.TypeNode != nil {
		varType = tc.resolveTypeNode(v.TypeNode)
		if varType == nil {
			tc.errorf(v.Name, "TC0014",
				"unknown type '%s'", v.TypeNode.Tok.Lexeme)
			varType = &CsType{Kind: TK_UNKNOWN, Name: "?"}
		}
	}

	if v.Init != nil {
		initType := tc.checkExpr(v.Init)
		if varType == nil {
			// var inference
			varType = initType
		} else {
			tc.expectAssignable(varType, initType, v.Name, "variable initializer")
		}
	}

	if varType == nil {
		varType = &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}

	if v.IsConst && v.Init == nil {
		tc.hint(v.Name, "TC0015",
			fmt.Sprintf("const variable '%s' must have an initializer", v.Name.Lexeme),
			"Add '= <value>' to the declaration")
	}

	if !tc.scope.define(&Symbol{Name: v.Name.Lexeme, Type: varType, Tok: v.Name, IsConst: v.IsConst}) {
		tc.errorf(v.Name, "TC0016",
			"variable '%s' is already declared in this scope", v.Name.Lexeme,
			"Use a different variable name or remove the duplicate declaration")
	}
}

func (tc *TypeChecker) checkIf(i *IfStmt) {
	condType := tc.checkExpr(i.Cond)
	if condType != nil && condType.Kind != TK_BOOL && condType.Kind != TK_UNKNOWN {
		tc.hint(i.Tok, "TC0017",
			"if condition must be of type 'bool'",
			fmt.Sprintf("Got type '%s', expected 'bool'", condType.String()),
			"Use a comparison operator: if (x == 0)")
	}
	tc.checkStmt(i.Then)
	if i.Else != nil {
		tc.checkStmt(i.Else)
	}
}

func (tc *TypeChecker) checkWhile(w *WhileStmt) {
	condType := tc.checkExpr(w.Cond)
	if condType != nil && condType.Kind != TK_BOOL && condType.Kind != TK_UNKNOWN {
		tc.warnf(w.Tok, "TC0018",
			"while condition should be of type 'bool', got '%s'", condType.String())
	}
	tc.inLoop++
	tc.checkStmt(w.Body)
	tc.inLoop--
}

func (tc *TypeChecker) checkDoWhile(d *DoWhileStmt) {
	tc.inLoop++
	tc.checkStmt(d.Body)
	tc.inLoop--
	tc.checkExpr(d.Cond)
}

func (tc *TypeChecker) checkFor(f *ForStmt) {
	outer := tc.scope
	tc.scope = newScope(outer)
	if f.Init != nil {
		tc.checkStmt(f.Init)
	}
	if f.Cond != nil {
		tc.checkExpr(f.Cond)
	}
	for _, p := range f.Post {
		tc.checkExpr(p)
	}
	tc.inLoop++
	tc.checkStmt(f.Body)
	tc.inLoop--
	tc.scope = outer
}

func (tc *TypeChecker) checkForeach(f *ForeachStmt) {
	elemType := tc.resolveTypeNode(f.ElemType)
	if elemType == nil {
		tc.errorf(f.ElemName, "TC0019",
			"unknown element type '%s' in foreach", f.ElemType.Tok.Lexeme)
		elemType = &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}
	tc.checkExpr(f.Range)
	outer := tc.scope
	tc.scope = newScope(outer)
	tc.scope.define(&Symbol{Name: f.ElemName.Lexeme, Type: elemType, Tok: f.ElemName})
	tc.inLoop++
	tc.checkStmt(f.Body)
	tc.inLoop--
	tc.scope = outer
}

func (tc *TypeChecker) checkSwitch(s *SwitchStmt) {
	tc.checkExpr(s.Expr)
	tc.inSwitch++
	for _, c := range s.Cases {
		if c.Value != nil {
			tc.checkExpr(c.Value)
		}
		for _, st := range c.Body {
			tc.checkStmt(st)
		}
	}
	tc.inSwitch--
}

func (tc *TypeChecker) checkThrow(t *ThrowStmt) {
	if t.Expr != nil {
		tc.checkExpr(t.Expr)
	}
}

func (tc *TypeChecker) checkTryCatch(t *TryCatchStmt) {
	tc.checkBlock(t.Body)
	for _, c := range t.Catches {
		outer := tc.scope
		tc.scope = newScope(outer)
		if c.ExcName.Lexeme != "" {
			excType := tc.resolveTypeNode(c.ExcType)
			if excType == nil {
				excType = &CsType{Kind: TK_CLASS, Name: "Exception"}
			}
			tc.scope.define(&Symbol{Name: c.ExcName.Lexeme, Type: excType, Tok: c.ExcName})
		}
		tc.checkBlock(c.Body)
		tc.scope = outer
	}
	if t.Finally != nil {
		tc.checkBlock(t.Finally)
	}
}

// ── Expression type inference ─────────────────────────────────────────────────

func (tc *TypeChecker) checkExpr(e Expr) *CsType {
	if e == nil {
		return &CsType{Kind: TK_VOID, Name: "void"}
	}
	switch v := e.(type) {
	case *IntLit:
		return tc.typeMap["int"]
	case *FloatLit:
		return tc.typeMap["float"]
	case *DoubleLit:
		return tc.typeMap["double"]
	case *StringLit:
		return tc.typeMap["string"]
	case *CharLit:
		return tc.typeMap["char"]
	case *BoolLit:
		return tc.typeMap["bool"]
	case *NullLit:
		return &CsType{Kind: TK_NULL, Name: "null"}
	case *IdentExpr:
		return tc.checkIdent(v)
	case *ThisExpr:
		sym := tc.scope.lookup("this")
		if sym != nil {
			return sym.Type
		}
		return &CsType{Kind: TK_UNKNOWN, Name: "?"}
	case *BaseExpr:
		return &CsType{Kind: TK_UNKNOWN, Name: "base"}
	case *BinaryExpr:
		return tc.checkBinary(v)
	case *UnaryExpr:
		return tc.checkUnary(v)
	case *AssignExpr:
		return tc.checkAssign(v)
	case *TernaryExpr:
		return tc.checkTernary(v)
	case *CallExpr:
		return tc.checkCall(v)
	case *MemberExpr:
		return tc.checkMember(v)
	case *IndexExpr:
		return tc.checkIndex(v)
	case *NewExpr:
		return tc.checkNew(v)
	case *NewArrayExpr:
		return tc.checkNewArray(v)
	case *CastExpr:
		return tc.checkCast(v)
	case *IsExpr:
		tc.checkExpr(v.Expr)
		return tc.typeMap["bool"]
	case *AsExpr:
		tc.checkExpr(v.Expr)
		t := tc.resolveTypeNode(v.Type)
		if t != nil {
			t2 := *t
			t2.IsNullable = true
			return &t2
		}
		return &CsType{Kind: TK_UNKNOWN, Name: "?"}
	case *TypeofExpr:
		return &CsType{Kind: TK_STRING, Name: "string"}
	case *NullCoalExpr:
		left := tc.checkExpr(v.Left)
		tc.checkExpr(v.Right)
		return left
	case *ArrayInitExpr:
		for _, el := range v.Elems {
			tc.checkExpr(el)
		}
		return &CsType{Kind: TK_ARRAY, Name: "array"}
	}
	return &CsType{Kind: TK_UNKNOWN, Name: "?"}
}

func (tc *TypeChecker) checkIdent(i *IdentExpr) *CsType {
	sym := tc.scope.lookup(i.Tok.Lexeme)
	if sym != nil {
		return sym.Type
	}
	// Could be a type name used in static context
	if t, ok := tc.typeMap[i.Tok.Lexeme]; ok {
		return t
	}
	// Lenient: don't error on unknown identifiers (could be from external libs)
	return &CsType{Kind: TK_UNKNOWN, Name: i.Tok.Lexeme}
}

func (tc *TypeChecker) checkBinary(b *BinaryExpr) *CsType {
	left := tc.checkExpr(b.Left)
	right := tc.checkExpr(b.Right)

	switch b.Op.Type {
	case TOKEN_PLUS, TOKEN_MINUS, TOKEN_STAR, TOKEN_SLASH, TOKEN_PERCENT:
		if left != nil && right != nil {
			if left.Kind == TK_STRING || right.Kind == TK_STRING {
				if b.Op.Type == TOKEN_PLUS {
					return tc.typeMap["string"]
				}
				if left.Kind == TK_STRING && b.Op.Type != TOKEN_PLUS {
					tc.errorf(b.Op, "TC0030",
						"operator '%s' cannot be applied to type 'string'", b.Op.Lexeme,
						"String concatenation uses '+', not other arithmetic operators")
				}
			}
			return tc.widenNumeric(left, right)
		}
	case TOKEN_LT, TOKEN_GT, TOKEN_LT_EQ, TOKEN_GT_EQ:
		return tc.typeMap["bool"]
	case TOKEN_EQ_EQ, TOKEN_BANG_EQ:
		return tc.typeMap["bool"]
	case TOKEN_AMP_AMP, TOKEN_PIPE_PIPE:
		if left != nil && left.Kind != TK_BOOL && left.Kind != TK_UNKNOWN {
			tc.errorf(b.Op, "TC0031",
				"operator '%s' requires bool operands, got '%s'", b.Op.Lexeme, left.String())
		}
		if right != nil && right.Kind != TK_BOOL && right.Kind != TK_UNKNOWN {
			tc.errorf(b.Op, "TC0031",
				"operator '%s' requires bool operands, got '%s'", b.Op.Lexeme, right.String())
		}
		return tc.typeMap["bool"]
	case TOKEN_AMP, TOKEN_PIPE, TOKEN_CARET, TOKEN_LSHIFT, TOKEN_RSHIFT:
		if left != nil && !left.IsIntegral() && left.Kind != TK_UNKNOWN {
			tc.errorf(b.Op, "TC0032",
				"bitwise operator '%s' requires integral operands", b.Op.Lexeme)
		}
		return left
	}
	if left != nil {
		return left
	}
	return &CsType{Kind: TK_UNKNOWN, Name: "?"}
}

func (tc *TypeChecker) widenNumeric(a, b *CsType) *CsType {
	if a.Kind == TK_UNKNOWN || b.Kind == TK_UNKNOWN {
		if a.Kind != TK_UNKNOWN {
			return a
		}
		return b
	}
	order := []CsTypeKind{TK_SBYTE, TK_BYTE, TK_SHORT, TK_USHORT, TK_INT, TK_UINT, TK_LONG, TK_ULONG, TK_FLOAT, TK_DOUBLE}
	ai, bi := -1, -1
	for i, k := range order {
		if a.Kind == k {
			ai = i
		}
		if b.Kind == k {
			bi = i
		}
	}
	if ai >= bi {
		return a
	}
	return b
}

func (tc *TypeChecker) checkUnary(u *UnaryExpr) *CsType {
	t := tc.checkExpr(u.Operand)
	switch u.Op.Type {
	case TOKEN_BANG:
		if t != nil && t.Kind != TK_BOOL && t.Kind != TK_UNKNOWN {
			tc.errorf(u.Op, "TC0033",
				"'!' operator requires bool operand, got '%s'", t.String())
		}
		return tc.typeMap["bool"]
	case TOKEN_MINUS, TOKEN_PLUS:
		if t != nil && !t.IsNumeric() && t.Kind != TK_UNKNOWN {
			tc.errorf(u.Op, "TC0034",
				"unary '%s' requires numeric operand, got '%s'", u.Op.Lexeme, t.String())
		}
		return t
	case TOKEN_TILDE:
		if t != nil && !t.IsIntegral() && t.Kind != TK_UNKNOWN {
			tc.errorf(u.Op, "TC0035",
				"'~' requires integral operand, got '%s'", t.String())
		}
		return t
	case TOKEN_PLUS_PLUS, TOKEN_MINUS_MINUS:
		if t != nil && !t.IsNumeric() && t.Kind != TK_UNKNOWN {
			tc.errorf(u.Op, "TC0036",
				"'%s' requires numeric operand, got '%s'", u.Op.Lexeme, t.String())
		}
		return t
	}
	return t
}

func (tc *TypeChecker) checkAssign(a *AssignExpr) *CsType {
	target := tc.checkExpr(a.Target)
	val := tc.checkExpr(a.Value)
	tc.expectAssignable(target, val, a.Op, "assignment")
	// check const assignment
	if ident, ok := a.Target.(*IdentExpr); ok {
		sym := tc.scope.lookup(ident.Tok.Lexeme)
		if sym != nil && sym.IsConst {
			tc.errorf(a.Op, "TC0037",
				"cannot assign to const variable '%s'", ident.Tok.Lexeme,
				"Remove the 'const' modifier to make the variable mutable")
		}
	}
	return target
}

func (tc *TypeChecker) checkTernary(t *TernaryExpr) *CsType {
	condType := tc.checkExpr(t.Cond)
	if condType != nil && condType.Kind != TK_BOOL && condType.Kind != TK_UNKNOWN {
		tc.errorf(t.QMark, "TC0038",
			"ternary condition must be bool, got '%s'", condType.String())
	}
	thenType := tc.checkExpr(t.Then)
	tc.checkExpr(t.Else)
	return thenType
}

func (tc *TypeChecker) checkCall(c *CallExpr) *CsType {
	tc.checkExpr(c.Callee)
	for _, arg := range c.Args {
		tc.checkExpr(arg)
	}
	// typeof() special handling
	if me, ok := c.Callee.(*IdentExpr); ok {
		if me.Tok.Lexeme == "typeof" {
			return tc.typeMap["string"]
		}
	}
	return &CsType{Kind: TK_UNKNOWN, Name: "?"}
}

/*
	func (tc *TypeChecker) checkMember(m *MemberExpr) *CsType {
		tc.checkExpr(m.Object)
		return &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}
*/
func (tc *TypeChecker) checkIndex(i *IndexExpr) *CsType {
	objType := tc.checkExpr(i.Object)
	idxType := tc.checkExpr(i.Index)
	if idxType != nil && !idxType.IsIntegral() && idxType.Kind != TK_UNKNOWN {
		tc.errorf(i.LBracket, "TC0039",
			"array index must be an integral type, got '%s'", idxType.String(),
			"Use int, long, or another integer type as array index")
	}
	if objType != nil && objType.Kind == TK_ARRAY && objType.ElemType != nil {
		return objType.ElemType
	}
	if objType != nil && objType.Kind == TK_STRING {
		return tc.typeMap["char"]
	}
	return &CsType{Kind: TK_UNKNOWN, Name: "?"}
}

func (tc *TypeChecker) checkNew(n *NewExpr) *CsType {
	t := tc.resolveTypeNode(n.Type)
	if t == nil {
		tc.errorf(n.Tok, "TC0040",
			"unknown type '%s' in 'new' expression", n.Type.Tok.Lexeme,
			"Make sure the class is declared or imported")
		return &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}
	for _, arg := range n.Args {
		tc.checkExpr(arg)
	}
	return t
}

func (tc *TypeChecker) checkNewArray(n *NewArrayExpr) *CsType {
	elemType := tc.resolveTypeNode(n.ElemType)
	if elemType == nil {
		tc.errorf(n.Tok, "TC0041",
			"unknown element type '%s' in array creation", n.ElemType.Tok.Lexeme)
		elemType = &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}
	if n.Size != nil {
		sizeType := tc.checkExpr(n.Size)
		if sizeType != nil && !sizeType.IsIntegral() && sizeType.Kind != TK_UNKNOWN {
			tc.errorf(n.Tok, "TC0042",
				"array size must be an integral type, got '%s'", sizeType.String())
		}
	}
	for _, el := range n.Init {
		tc.checkExpr(el)
	}
	return &CsType{Kind: TK_ARRAY, Name: elemType.Name + "[]", ElemType: elemType}
}

func (tc *TypeChecker) checkCast(c *CastExpr) *CsType {
	src := tc.checkExpr(c.Expr)
	dst := tc.resolveTypeNode(c.Type)
	if dst == nil {
		tc.errorf(c.LParen, "TC0043",
			"unknown cast target type '%s'", c.Type.Tok.Lexeme)
		return &CsType{Kind: TK_UNKNOWN, Name: "?"}
	}
	// Warn on obviously wrong casts
	if src != nil && src.Kind != TK_UNKNOWN && dst.Kind != TK_UNKNOWN {
		if src.Kind == TK_BOOL && dst.IsNumeric() {
			tc.warnf(c.LParen, "TC0201",
				"casting bool to numeric is unconventional",
				"Consider using a ternary: (x ? 1 : 0)")
		}
		if src.Kind == TK_STRING && dst.IsNumeric() {
			tc.hint(c.LParen, "TC0202",
				"cannot cast string to numeric directly",
				fmt.Sprintf("Use %s.Parse() for conversion", dst.Name))
		}
	}
	return dst
}

// ── Type resolution ───────────────────────────────────────────────────────────

func (tc *TypeChecker) resolveTypeNode(tn *TypeNode) *CsType {
	if tn == nil {
		return nil
	}
	base := tc.typeMap[tn.Tok.Lexeme]
	if base == nil {
		return nil
	}
	if tn.IsArray {
		return &CsType{
			Kind:       TK_ARRAY,
			Name:       base.Name + "[]",
			ElemType:   base,
			IsNullable: tn.IsNullable,
		}
	}
	if tn.IsNullable {
		t := *base
		t.IsNullable = true
		return &t
	}
	if len(tn.Generic) > 0 {
		args := make([]*CsType, len(tn.Generic))
		for i, g := range tn.Generic {
			args[i] = tc.resolveTypeNode(g)
		}
		return &CsType{Kind: TK_GENERIC, Name: base.Name, TypeArgs: args, Decl: base.Decl}
	}
	return base
}

// expectAssignable emits an error if src is not assignable to dst
func (tc *TypeChecker) expectAssignable(dst, src *CsType, tok Token, ctx string) {
	if dst == nil || src == nil {
		return
	}
	if dst.Kind == TK_UNKNOWN || src.Kind == TK_UNKNOWN {
		return
	}
	if src.Kind == TK_NULL {
		if !dst.IsNullable && dst.Kind != TK_STRING && dst.Kind != TK_OBJECT &&
			dst.Kind != TK_CLASS && dst.Kind != TK_ARRAY {
			tc.hint(tok, "TC0050",
				fmt.Sprintf("cannot assign null to non-nullable type '%s' in %s", dst.String(), ctx),
				"Declare the type as nullable: "+dst.Name+"?")
		}
		return
	}
	if dst.Name == src.Name {
		return
	}
	if dst.Kind == TK_OBJECT {
		return // everything is assignable to object
	}
	if dst.IsNumeric() && src.IsNumeric() {
		// allow widening
		return
	}
	if dst.Kind == TK_DOUBLE && src.Kind == TK_FLOAT {
		return
	}
	if dst.Kind == TK_STRING && src.Kind == TK_STRING {
		return
	}
	// For class/interface types we can't do full inheritance checks without full symbol resolution
	if dst.Kind == TK_CLASS || dst.Kind == TK_INTERFACE {
		return
	}
	if src.Kind == TK_CLASS || src.Kind == TK_STRUCT {
		return
	}
	if dst.Kind != src.Kind {
		tc.hint(tok, "TC0051",
			fmt.Sprintf("type mismatch in %s: expected '%s', got '%s'", ctx, dst.String(), src.String()),
			fmt.Sprintf("Cast the value: (%s)value", dst.Name),
			"Or convert using a conversion method")
	}
}
