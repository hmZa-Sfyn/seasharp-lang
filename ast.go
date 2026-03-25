package main

// ── AST Node Hierarchy ─────────────────────────────────────────────────────────

type Node interface {
	nodePos() Token
}

type Expr interface {
	Node
	exprNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Decl interface {
	Node
	declNode()
}

// ── Annotations ─────────────────────────────────────────────────────────────

type Annotation struct {
	At   Token
	Name Token
	Args []Expr
}

// ── Type Nodes ───────────────────────────────────────────────────────────────

type TypeNode struct {
	Tok        Token
	IsArray    bool
	ArrayRank  int
	IsNullable bool
	Generic    []*TypeNode
}

func (t *TypeNode) nodePos() Token { return t.Tok }

// ── Top-Level ────────────────────────────────────────────────────────────────

type CompilationUnit struct {
	Usings     []*UsingDecl
	Namespaces []*NamespaceDecl
	Members    []Decl
}

type UsingDecl struct {
	Tok  Token
	Path []Token
}

func (u *UsingDecl) nodePos() Token { return u.Tok }
func (u *UsingDecl) declNode()      {}

type NamespaceDecl struct {
	Tok     Token
	Path    []Token
	Members []Decl
}

func (n *NamespaceDecl) nodePos() Token { return n.Tok }
func (n *NamespaceDecl) declNode()      {}

// ── Class / Struct / Interface ────────────────────────────────────────────────

type AccessMod int

const (
	ACCESS_DEFAULT AccessMod = iota
	ACCESS_PUBLIC
	ACCESS_PRIVATE
	ACCESS_PROTECTED
	ACCESS_INTERNAL
)

type ClassDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	IsStatic    bool
	IsAbstract  bool
	IsSealed    bool
	Tok         Token
	Name        Token
	TypeParams  []Token
	BaseTypes   []*TypeNode
	Members     []Decl
}

func (c *ClassDecl) nodePos() Token { return c.Tok }
func (c *ClassDecl) declNode()      {}

type StructDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	Tok         Token
	Name        Token
	TypeParams  []Token
	BaseTypes   []*TypeNode
	Members     []Decl
}

func (s *StructDecl) nodePos() Token { return s.Tok }
func (s *StructDecl) declNode()      {}

type InterfaceDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	Tok         Token
	Name        Token
	TypeParams  []Token
	BaseTypes   []*TypeNode
	Members     []Decl
}

func (i *InterfaceDecl) nodePos() Token { return i.Tok }
func (i *InterfaceDecl) declNode()      {}

type EnumDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	Tok         Token
	Name        Token
	BaseType    *TypeNode
	Members     []*EnumMember
}

func (e *EnumDecl) nodePos() Token { return e.Tok }
func (e *EnumDecl) declNode()      {}

type EnumMember struct {
	Annotations []*Annotation
	Name        Token
	Value       Expr
}

// ── Members ───────────────────────────────────────────────────────────────────

type FieldDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	IsStatic    bool
	IsReadonly  bool
	IsConst     bool
	Type        *TypeNode
	Name        Token
	Init        Expr
}

func (f *FieldDecl) nodePos() Token { return f.Name }
func (f *FieldDecl) declNode()      {}

type PropertyDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	IsStatic    bool
	IsAbstract  bool
	IsVirtual   bool
	IsOverride  bool
	Type        *TypeNode
	Name        Token
	Getter      *PropertyAccessor
	Setter      *PropertyAccessor
}

func (p *PropertyDecl) nodePos() Token { return p.Name }
func (p *PropertyDecl) declNode()      {}

type PropertyAccessor struct {
	Tok    Token
	Access AccessMod
	Body   *Block
}

type MethodDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	IsStatic    bool
	IsAbstract  bool
	IsVirtual   bool
	IsOverride  bool
	IsExtern    bool
	ReturnType  *TypeNode
	Name        Token
	TypeParams  []Token
	Params      []*Param
	Body        *Block
}

func (m *MethodDecl) nodePos() Token { return m.Name }
func (m *MethodDecl) declNode()      {}

type ConstructorDecl struct {
	Annotations []*Annotation
	Access      AccessMod
	Name        Token
	Params      []*Param
	BaseArgs    []Expr
	Body        *Block
}

func (c *ConstructorDecl) nodePos() Token { return c.Name }
func (c *ConstructorDecl) declNode()      {}

type DestructorDecl struct {
	Tok  Token
	Name Token
	Body *Block
}

func (d *DestructorDecl) nodePos() Token { return d.Tok }
func (d *DestructorDecl) declNode()      {}

type Param struct {
	Annotations []*Annotation
	Modifier    Token
	Type        *TypeNode
	Name        Token
	Default     Expr
}

// ── Statements ───────────────────────────────────────────────────────────────

type Block struct {
	LBrace Token
	Stmts  []Stmt
	RBrace Token
}

func (b *Block) nodePos() Token { return b.LBrace }
func (b *Block) stmtNode()      {}

type ExprStmt struct {
	Expr Expr
	Semi Token
}

func (e *ExprStmt) nodePos() Token { return e.Expr.nodePos() }
func (e *ExprStmt) stmtNode()      {}

type ReturnStmt struct {
	Tok   Token
	Value Expr
}

func (r *ReturnStmt) nodePos() Token { return r.Tok }
func (r *ReturnStmt) stmtNode()      {}

type BreakStmt struct{ Tok Token }

func (b *BreakStmt) nodePos() Token { return b.Tok }
func (b *BreakStmt) stmtNode()      {}

type ContinueStmt struct{ Tok Token }

func (c *ContinueStmt) nodePos() Token { return c.Tok }
func (c *ContinueStmt) stmtNode()      {}

type ThrowStmt struct {
	Tok  Token
	Expr Expr
}

func (t *ThrowStmt) nodePos() Token { return t.Tok }
func (t *ThrowStmt) stmtNode()      {}

type LocalVarDecl struct {
	IsConst  bool
	Type     Token
	TypeNode *TypeNode
	Name     Token
	Init     Expr
}

func (l *LocalVarDecl) nodePos() Token { return l.Name }
func (l *LocalVarDecl) stmtNode()      {}

type IfStmt struct {
	Tok     Token
	Cond    Expr
	Then    Stmt
	ElseTok Token
	Else    Stmt
}

func (i *IfStmt) nodePos() Token { return i.Tok }
func (i *IfStmt) stmtNode()      {}

type WhileStmt struct {
	Tok  Token
	Cond Expr
	Body Stmt
}

func (w *WhileStmt) nodePos() Token { return w.Tok }
func (w *WhileStmt) stmtNode()      {}

type DoWhileStmt struct {
	Tok  Token
	Body Stmt
	Cond Expr
}

func (d *DoWhileStmt) nodePos() Token { return d.Tok }
func (d *DoWhileStmt) stmtNode()      {}

type ForStmt struct {
	Tok  Token
	Init Stmt
	Cond Expr
	Post []Expr
	Body Stmt
}

func (f *ForStmt) nodePos() Token { return f.Tok }
func (f *ForStmt) stmtNode()      {}

type ForeachStmt struct {
	Tok      Token
	ElemType *TypeNode
	ElemName Token
	Range    Expr
	Body     Stmt
}

func (f *ForeachStmt) nodePos() Token { return f.Tok }
func (f *ForeachStmt) stmtNode()      {}

type SwitchStmt struct {
	Tok   Token
	Expr  Expr
	Cases []*SwitchCase
}

func (s *SwitchStmt) nodePos() Token { return s.Tok }
func (s *SwitchStmt) stmtNode()      {}

type SwitchCase struct {
	Tok   Token
	Value Expr
	Body  []Stmt
}

type TryCatchStmt struct {
	Tok     Token
	Body    *Block
	Catches []*CatchClause
	Finally *Block
}

func (t *TryCatchStmt) nodePos() Token { return t.Tok }
func (t *TryCatchStmt) stmtNode()      {}

type CatchClause struct {
	Tok     Token
	ExcType *TypeNode
	ExcName Token
	Body    *Block
}

// ── Expressions ──────────────────────────────────────────────────────────────

type IntLit struct {
	Tok Token
	Val int64
}

func (i *IntLit) nodePos() Token { return i.Tok }
func (i *IntLit) exprNode()      {}

type FloatLit struct {
	Tok Token
	Val float64
}

func (f *FloatLit) nodePos() Token { return f.Tok }
func (f *FloatLit) exprNode()      {}

type DoubleLit struct {
	Tok Token
	Val float64
}

func (d *DoubleLit) nodePos() Token { return d.Tok }
func (d *DoubleLit) exprNode()      {}

type StringLit struct {
	Tok Token
	Val string
}

func (s *StringLit) nodePos() Token { return s.Tok }
func (s *StringLit) exprNode()      {}

type CharLit struct {
	Tok Token
	Val rune
}

func (c *CharLit) nodePos() Token { return c.Tok }
func (c *CharLit) exprNode()      {}

type BoolLit struct {
	Tok Token
	Val bool
}

func (b *BoolLit) nodePos() Token { return b.Tok }
func (b *BoolLit) exprNode()      {}

type NullLit struct{ Tok Token }

func (n *NullLit) nodePos() Token { return n.Tok }
func (n *NullLit) exprNode()      {}

type IdentExpr struct{ Tok Token }

func (i *IdentExpr) nodePos() Token { return i.Tok }
func (i *IdentExpr) exprNode()      {}

type ThisExpr struct{ Tok Token }

func (t *ThisExpr) nodePos() Token { return t.Tok }
func (t *ThisExpr) exprNode()      {}

type BaseExpr struct{ Tok Token }

func (b *BaseExpr) nodePos() Token { return b.Tok }
func (b *BaseExpr) exprNode()      {}

type UnaryExpr struct {
	Op      Token
	Operand Expr
	IsPost  bool
}

func (u *UnaryExpr) nodePos() Token { return u.Op }
func (u *UnaryExpr) exprNode()      {}

type BinaryExpr struct {
	Left  Expr
	Op    Token
	Right Expr
}

func (b *BinaryExpr) nodePos() Token { return b.Left.nodePos() }
func (b *BinaryExpr) exprNode()      {}

type AssignExpr struct {
	Target Expr
	Op     Token
	Value  Expr
}

func (a *AssignExpr) nodePos() Token { return a.Target.nodePos() }
func (a *AssignExpr) exprNode()      {}

type TernaryExpr struct {
	Cond  Expr
	Then  Expr
	Else  Expr
	QMark Token
}

func (t *TernaryExpr) nodePos() Token { return t.Cond.nodePos() }
func (t *TernaryExpr) exprNode()      {}

type CallExpr struct {
	Callee   Expr
	LParen   Token
	TypeArgs []*TypeNode
	Args     []Expr
}

func (c *CallExpr) nodePos() Token { return c.Callee.nodePos() }
func (c *CallExpr) exprNode()      {}

type MemberExpr struct {
	Object Expr
	Dot    Token
	Member Token
}

func (m *MemberExpr) nodePos() Token { return m.Object.nodePos() }
func (m *MemberExpr) exprNode()      {}

type IndexExpr struct {
	Object   Expr
	LBracket Token
	Index    Expr
}

func (i *IndexExpr) nodePos() Token { return i.Object.nodePos() }
func (i *IndexExpr) exprNode()      {}

type NewExpr struct {
	Tok  Token
	Type *TypeNode
	Args []Expr
}

func (n *NewExpr) nodePos() Token { return n.Tok }
func (n *NewExpr) exprNode()      {}

type NewArrayExpr struct {
	Tok      Token
	ElemType *TypeNode
	Size     Expr
	Init     []Expr
}

func (n *NewArrayExpr) nodePos() Token { return n.Tok }
func (n *NewArrayExpr) exprNode()      {}

type CastExpr struct {
	LParen Token
	Type   *TypeNode
	Expr   Expr
}

func (c *CastExpr) nodePos() Token { return c.LParen }
func (c *CastExpr) exprNode()      {}

type IsExpr struct {
	Expr Expr
	Tok  Token
	Type *TypeNode
}

func (i *IsExpr) nodePos() Token { return i.Expr.nodePos() }
func (i *IsExpr) exprNode()      {}

type AsExpr struct {
	Expr Expr
	Tok  Token
	Type *TypeNode
}

func (a *AsExpr) nodePos() Token { return a.Expr.nodePos() }
func (a *AsExpr) exprNode()      {}

type TypeofExpr struct {
	Tok  Token
	Type *TypeNode
}

func (t *TypeofExpr) nodePos() Token { return t.Tok }
func (t *TypeofExpr) exprNode()      {}

type NullCoalExpr struct {
	Left  Expr
	Op    Token
	Right Expr
}

func (n *NullCoalExpr) nodePos() Token { return n.Left.nodePos() }
func (n *NullCoalExpr) exprNode()      {}

type ArrayInitExpr struct {
	LBrace Token
	Elems  []Expr
}

func (a *ArrayInitExpr) nodePos() Token { return a.LBrace }
func (a *ArrayInitExpr) exprNode()      {}
