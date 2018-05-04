package compiler

import (
	"fmt"

	"github.com/srax47/godzilla/ast"
	"github.com/srax47/godzilla/runtime"
	"github.com/srax47/godzilla/source"
	"github.com/srax47/godzilla/utils"
)

func Compile(f *ast.File) *source.Code {
	code := source.NewCode()

	c := &compiler{
		code: code,
		ctx:  runtime.NewDefaultContext(),
	}
	c.compile(f)

	return code
}

type compiler struct {
	code *source.Code
	ctx  *runtime.Context
}

func (c *compiler) compile(f *ast.File) {
	c.compileProgram(f.Program)
}

func (c *compiler) compileProgram(p *ast.Program) {
	for _, s := range p.Body {
		c.writeLineNo(s)
		c.compileStatement(s)
		c.code.WriteLine("")
	}
}

// statements

func (c *compiler) compileStatement(s ast.Statement) {
	switch v := s.(type) {
	case *ast.ExpressionStatement:
		c.compileExpressionStatement(v)
	case *ast.VariableDeclaration:
		c.compileVariableDeclaration(v)
	default:
		panic("unknown statement type " + utils.TypeOf(v))
	}
}

func (c *compiler) compileExpressionStatement(es *ast.ExpressionStatement) {
	c.compileExpression(es.Expression)
}

// TODO: ignore Kind for now
func (c *compiler) compileVariableDeclaration(vd *ast.VariableDeclaration) {
	for _, d := range vd.Declarations {
		c.compileVariableDeclarator(d)
	}
}

func (c *compiler) compileVariableDeclarator(vd *ast.VariableDeclarator) {
	name := vd.ID.Name

	c.code.WriteLine(fmt.Sprintf("var %s Object", name))
	c.code.WriteLine(fmt.Sprintf("_ = %s", name))
	if vd.Init != nil {
		c.code.Write(fmt.Sprintf("%s = ", name))
		c.compileExpression(vd.Init)
		c.code.WriteLine("")
	}
	c.code.Write(fmt.Sprintf(`global.DefineProperty("%s", %s)`, name, name))
	c.defineVar(name)
}

// expressions

func (c *compiler) compileExpression(e ast.Expression) {
	switch v := e.(type) {
	case *ast.CallExpression:
		c.compileCallExpression(v)
	case *ast.AssignmentExpression:
		c.compileAssignmentExpression(v)
	case *ast.BinaryExpression:
		c.compileBinaryExpression(v)
	case *ast.MemberExpression:
		c.compileMemberExpression(v)
	case *ast.Identifier:
		c.compileIdentifier(v)
	case *ast.StringLiteral:
		c.compileStringLiteral(v)
	case *ast.NumericLiteral:
		c.compileNumericLiteral(v)
	default:
		panic("unknown expression type " + utils.TypeOf(v))
	}
}

func (c *compiler) compileCallExpression(ce *ast.CallExpression) {
	c.compileExpression(ce.Callee)
	c.code.Write("([]Object{")
	for i, arg := range ce.Arguments {
		c.compileExpression(arg)
		if i != len(ce.Arguments)-1 {
			c.code.Write(", ")
		}
	}
	c.code.Write("})\n")
}

// TODO: ignoring computed value for now
func (c *compiler) compileMemberExpression(me *ast.MemberExpression) {
	if me.Computed {
		panic("computed MemberExpression is not supported")
	}

	if builtInFunc := c.getBuiltinFunc(me.Object, me.Property); builtInFunc == "" {
		c.compileExpression(me.Object)
		c.code.Write(".")
		c.compileExpression(me.Property)
	} else {
		c.code.Write(builtInFunc)
	}
}

func (c *compiler) compileAssignmentExpression(ae *ast.AssignmentExpression) {
	c.compileExpression(ae.Left)
	c.code.Write(fmt.Sprintf(" %s ", ae.Operator))
	c.compileExpression(ae.Right)
}

func (c *compiler) compileBinaryExpression(be *ast.BinaryExpression) {
	c.compileExpression(be.Left)
	c.code.Write(fmt.Sprintf(" %s ", be.Operator))
	c.compileExpression(be.Right)
}

func (c *compiler) compileIdentifier(i *ast.Identifier) {
	if c.isVarDefined(i.Name) {
		c.code.Write(i.Name)
	} else {
		c.code.Write(fmt.Sprintf(`global.GetProperty("%s")`, i.Name))
	}
}

func (c *compiler) compileStringLiteral(s *ast.StringLiteral) {
	c.code.Write(fmt.Sprintf(`JSString("%s")`, s.Value))
}

func (c *compiler) compileNumericLiteral(n *ast.NumericLiteral) {
	c.code.Write(fmt.Sprintf(`JSNumber(%f)`, n.Value))
}

func (c *compiler) writeLineNo(node ast.Node) {
	c.code.WriteLine(fmt.Sprintf(`// line %d: %s`, node.GetAttr().Loc.Start.Line, node))
}

// defineVar defines the var when the compiler sees it
// This is used for optimizing compiled code for direct reference of var
// FIXME: defined var is cached in global context for now
func (c *compiler) defineVar(name string) {
	// FIXME: Prop value is a dummpy obj for now
	c.ctx.Global.DefineProperty(name, &runtime.JSObject{})
}

// FIXME: Using global context for now
func (c *compiler) isVarDefined(name string) bool {
	_, err := c.ctx.Global.GetProperty(name)
	return err == nil
}

func (c *compiler) getBuiltinFunc(objExp, propExp ast.Expression) string {
	oID, ok := objExp.(*ast.Identifier)
	if !ok {
		return ""
	}

	pID := propExp.(*ast.Identifier)
	if !ok {
		return ""
	}

	obj, err := c.ctx.Global.GetProperty(oID.Name)
	if err != nil || obj.Type() != runtime.JS_OBJECT_TYPE_OBJECT {
		return ""
	}

	prop, err := (obj.(*runtime.JSObject)).GetProperty(pID.Name)
	if err != nil || prop.Type() != runtime.JS_OBJECT_TYPE_FUNCTION {
		return ""
	}

	return (prop.(*runtime.JSFunction)).FuncName()
}
