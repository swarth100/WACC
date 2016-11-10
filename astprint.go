package main

import (
	"fmt"
	"strconv"
)

//
func (i IntType) ASTString(indent string) string {
	return addType(indent, "int")
}

//
func (b BoolType) ASTString(indent string) string {
	return addType(indent, "bool")
}

//
func (c CharType) ASTString(indent string) string {
	return addType(indent, "char")
}

func (p PairType) ASTString(indent string) string {
	var first string = "pair"
	var second string = "pair"

	if p.first != nil {
		first = fmt.Sprintf("%v", p.first.ASTString(indent))
	}
	if p.second != nil {
		second = fmt.Sprintf("%v", p.second.ASTString(indent))
	}
	return fmt.Sprintf("pair(%v, %v)", first, second)
}

func (a ArrayType) ASTString(indent string) string {
	var typeStats string = fmt.Sprintf("%v[]", a.base)
	return addType(indent, typeStats)
}

func (stmt SkipStatement) ASTString(indent string) string {
	return addIndAndNewLine(indent, "SKIP")
}

func (stmt BlockStatement) ASTString(indent string) string {
	return fmt.Sprintf("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXx")
}

//
func (stmt DeclareAssignStatement) ASTString(indent string) string {

	var declareStats string = fmt.Sprintf("%vDECLARE\n", addMinToIndent(indent))
	var innerIndent string = getGreaterIndent(indent)
	var lhsIndent string = addDoubleIndent(innerIndent, "LHS", stmt.ident)
	var rhsIndent string = addIndentForFirst(innerIndent, "RHS", stmt.rhs.ASTString(getGreaterIndent(innerIndent)))

	return fmt.Sprintf("%v%v%v%v", declareStats, stmt.waccType.ASTString(innerIndent), lhsIndent, rhsIndent)
}

func (lhs PairElemLHS) ASTString(indent string) string {
	if lhs.snd {
		return fmt.Sprintf("snd %v", lhs.expr.ASTString(indent))
	} else {
		return fmt.Sprintf("fst %v", lhs.expr.ASTString(indent))
	}
}

func (lhs ArrayLHS) ASTString(indent string) string {
	var indexes string

	for _, index := range lhs.index {
		indexes = fmt.Sprintf("%v[%v]", indexes, index)
	}

	return fmt.Sprintf("%v%v", lhs.ident, indexes)
}

func (lhs VarLHS) ASTString(indent string) string {
	return addIndAndNewLine(indent, lhs.ident)
}

func (rhs PairLiterRHS) ASTString(indent string) string {
	return fmt.Sprintf("newpair(%v, %v)", rhs.fst.ASTString(indent), rhs.snd.ASTString(indent))
}

func (rhs ArrayLiterRHS) ASTString(indent string) string {
	elemArr := []string{}

	for _, element := range rhs.elements {
		elemArr = append(elemArr, element.ASTString(indent))
	}

	return addArrayIndent(indent, "ARRAY LITERAL", elemArr)
}

func (rhs PairElemRHS) ASTString(indent string) string {
	if rhs.snd {
		return fmt.Sprintf("snd %v", rhs.expr.ASTString(indent))
	} else {
		return fmt.Sprintf("fst %v", rhs.expr.ASTString(indent))
	}
}

func (rhs FunctionCallRHS) ASTString(indent string) string {
	var innerStats string
	var nameStats string = addIndAndNewLine(indent, rhs.ident)

	for _, param := range rhs.args {
		innerStats = fmt.Sprintf("%v%v", innerStats, param.ASTString(indent))
	}

	return fmt.Sprintf("%v%v", nameStats, innerStats)

	//return fmt.Sprintf("call %v(%v)", rhs.ident, params)
}

func (lpar ExprLPar) ASTString(indent string) string {
	return ""
}

func (rpar ExprRPar) ASTString(indent string) string {
	return ""
}

func (rhs ExpressionRHS) ASTString(indent string) string {
	return rhs.expr.ASTString(indent)
	//return fmt.Sprintf("%v", rhs.expr.ASTString(indent))
}

func (stmt AssignStatement) ASTString(indent string) string {
	var declareStats string = fmt.Sprintf("%vASSIGNMENT\n", addMinToIndent(indent))
	var innerIndent string = getGreaterIndent(indent)
	var lhsIndent string = addIndentForFirst(innerIndent, "LHS", stmt.target.ASTString(getGreaterIndent(innerIndent)))
	var rhsIndent string = addIndentForFirst(innerIndent, "RHS", stmt.rhs.ASTString(getGreaterIndent(innerIndent)))

	return fmt.Sprintf("%v%v%v", declareStats, lhsIndent, rhsIndent)
}

func (stmt ReadStatement) ASTString(indent string) string {
	return fmt.Sprintf("%vread %v", addMinToIndent(indent), stmt.target.ASTString(indent))
}

func (stmt FreeStatement) ASTString(indent string) string {
	return fmt.Sprintf("%vfree %v", addMinToIndent(indent), stmt.expr.ASTString(indent))
}

func (ret ReturnStatement) ASTString(indent string) string {
	return addIndentForFirst(indent, "RETURN", ret.expr.ASTString(getGreaterIndent(indent)))
	//return fmt.Sprintf("%vreturn %v",  addMinToIndent(indent), ret.expr.ASTString(indent))
}

func (stmt ExitStatement) ASTString(indent string) string {
	return fmt.Sprintf("%vexit %v", addMinToIndent(indent), stmt.expr.ASTString(indent))
}

//
func (stmt PrintLnStatement) ASTString(indent string) string {
	return addIndentForFirst(indent, "PRINTLN", stmt.expr.ASTString(getGreaterIndent(indent)))
}

func (stmt PrintStatement) ASTString(indent string) string {
	return addIndentForFirst(indent, "PRINT", stmt.expr.ASTString(getGreaterIndent(indent)))
}

//
func (stmt IfStatement) ASTString(indent string) string {
	var stmtStats string
	var trueStats string
	var falseStats string
	var ifStats string
	var condStats string
	var thenStats string
	var elseStats string
	var innerIndent string = fmt.Sprintf("%v%v", indent, basicIndent)
	var doubleInnerIndent string = fmt.Sprintf("%v%v", innerIndent, basicIndent)

	ifStats = fmt.Sprintf("%vIF\n", addMinToIndent(indent))
	condStats = fmt.Sprintf("%vCONDITION\n", addMinToIndent(innerIndent))
	thenStats = fmt.Sprintf("%vTHEN\n", addMinToIndent(innerIndent))
	elseStats = fmt.Sprintf("%vELSE\n", addMinToIndent(innerIndent))

	stmtStats = stmt.cond.ASTString(doubleInnerIndent)

	st := stmt.trueStat
	for st.GetNext() != nil {
		trueStats = st.ASTString(doubleInnerIndent)
		st = st.GetNext()
	}

	trueStats = st.ASTString(doubleInnerIndent)

	st = stmt.falseStat
	for st.GetNext() != nil {
		falseStats = st.ASTString(doubleInnerIndent)
		st = st.GetNext()
	}

	falseStats = st.ASTString(doubleInnerIndent)

	return fmt.Sprintf("%v%v%v%v%v%v%v", ifStats, condStats, stmtStats, thenStats, trueStats, elseStats, falseStats)
}

func (stmt WhileStatement) ASTString(indent string) string {
	var body string
	var doStats string
	var innerIndent string = fmt.Sprintf("%v%v", indent, basicIndent)

	doStats = addIndAndNewLine(innerIndent, "DO")

	var condStats string = addIndentForFirst(innerIndent, "CONDITION", stmt.cond.ASTString(getGreaterIndent(innerIndent)))

	st := stmt.body
	for st.GetNext() != nil {
		body = st.ASTString(getGreaterIndent(innerIndent))
		doStats = fmt.Sprintf("%v%v", doStats, body)
		st = st.GetNext()
	}
	body = st.ASTString(getGreaterIndent(innerIndent))
	doStats = fmt.Sprintf("%v%v", doStats, body)

	var loopStats string = addIndAndNewLine(indent, "LOOP")

	return fmt.Sprintf("%v%v%v", loopStats, condStats, doStats)
}

func (fp FunctionParam) ASTString(indent string) string {
	return fmt.Sprintf("%v %v", fp.waccType, fp.name)
}

func (fd FunctionDef) ASTString(indent string) string {

	var params string
	var body string

	innerIndent := fmt.Sprintf("%v%v", indent, basicIndent)

	if len(fd.params) > 0 {
		params = fmt.Sprintf("%v", fd.params[0])

		for _, param := range fd.params[1:] {
			params = fmt.Sprintf("%v, %v", params, param)
		}
	}

	declaration := addIndAndNewLine(indent, fmt.Sprintf("%v %v(%v)", fd.returnType, fd.ident, params))

	st := fd.body
	for st.GetNext() != nil {
		body = fmt.Sprintf("%v%v", body, st.ASTString(innerIndent))
		st = st.GetNext()
	}

	body = fmt.Sprintf("%v%v", body, st.ASTString(innerIndent))

	return fmt.Sprintf("%v%v", declaration, body)
}

func (ident Ident) ASTString(indent string) string {
	return addIndAndNewLine(indent, ident.ident)
}

func (liter IntLiteral) ASTString(indent string) string {
	return addIndAndNewLine(indent, strconv.Itoa(liter.value))
}

//
func (liter BoolLiteralTrue) ASTString(indent string) string {
	return addIndAndNewLine(indent, "true")
}

//
func (liter BoolLiteralFalse) ASTString(indent string) string {
	return addIndAndNewLine(indent, "false")
}

func (liter CharLiteral) ASTString(indent string) string {
	var tmpStats string = fmt.Sprintf("'%v'", liter.char)
	return addIndAndNewLine(indent, tmpStats)
}

func (liter StringLiteral) ASTString(indent string) string {
	var tmp string = fmt.Sprintf("\"%v\"", liter.str)
	return addIndAndNewLine(indent, tmp)
}

func (liter PairLiteral) ASTString(indent string) string {
	return fmt.Sprintf("pair(%v, %v)", liter.fst, liter.snd)
}

func (liter NullPair) ASTString(indent string) string {
	return fmt.Sprintf("null")
}

func (elem ArrayElem) ASTString(indent string) string {
	var indexes string

	for _, index := range elem.indexes {
		indexes = fmt.Sprintf("%v[%v]", indexes, index)
	}

	return fmt.Sprintf("%v%v", elem.ident, indexes)
}

func (op UnaryOperatorNot) ASTString(indent string) string {
	return addIndentForFirst(indent, "!", op.GetExpression().ASTString(getGreaterIndent(indent)))
}

func (op UnaryOperatorNegate) ASTString(indent string) string {
	return addIndentForFirst(indent, "-", op.GetExpression().ASTString(getGreaterIndent(indent)))
}

func (op UnaryOperatorLen) ASTString(indent string) string {
	return addIndentForFirst(indent, "len", op.GetExpression().ASTString(getGreaterIndent(indent)))
}

func (op UnaryOperatorOrd) ASTString(indent string) string {
	return addIndentForFirst(indent, "ord", op.GetExpression().ASTString(getGreaterIndent(indent)))
}

func (op UnaryOperatorChr) ASTString(indent string) string {
	return addIndentForFirst(indent, "chr", op.GetExpression().ASTString(getGreaterIndent(indent)))
	//return fmt.Sprintf("chr %v", op.GetExpression())
}

//
func (op BinaryOperatorMult) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "*", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorDiv) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "/", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorMod) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "%%", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorAdd) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "+", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorSub) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "-", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorGreaterThan) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, ">", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorGreaterEqual) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, ">=", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorLessThan) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "<", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorLessEqual) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "<=", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorEqual) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "==", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

//
func (op BinaryOperatorNotEqual) ASTString(indent string) string {
	return addTripleIndentOnlyFst(indent, "!=", op.GetLHS().ASTString(getGreaterIndent(indent)), op.GetRHS().ASTString(getGreaterIndent(indent)))
}

func (op BinaryOperatorAnd) ASTString(indent string) string {
	return fmt.Sprintf("%v && %v", op.GetLHS(), op.GetRHS())
}

func (op BinaryOperatorOr) ASTString(indent string) string {
	return fmt.Sprintf("%v || %v", op.GetLHS(), op.GetRHS())
}

func addMinToIndent(indent string) string {
	return (indent + "- ")
}

func addAtGreaterIndent(indent string, value string) string {
	return fmt.Sprintf("%v%v\n", addMinToIndent(indent+basicIndent), value)
}

func addIndAndNewLine(indent string, value string) string {
	return fmt.Sprintf("%v%v\n", addMinToIndent(indent), value)
}

func addIndentForFirst(indent string, a1 string, a2 string) string {
	return fmt.Sprintf("%v%v", addIndAndNewLine(indent, a1), a2)
}

func addDoubleIndent(indent string, a1 string, a2 string) string {
	return addTripleIndent(indent, a1, a2, "")
}

func addArrayIndent(indent string, a1 string, arr []string) string {
	var innerIndent string = fmt.Sprintf("%v%v", indent, basicIndent)
	var innerStats string = ""

	var typeStats string = addIndAndNewLine(indent, a1)
	for _, element := range arr {
		innerStats = fmt.Sprintf("%v%v", innerStats, addIndAndNewLine(innerIndent, element))
	}

	return fmt.Sprintf("%v%v", typeStats, innerStats)
}

func getGreaterIndent(indent string) string {
	return fmt.Sprintf("%v%v", indent, basicIndent)
}

func addType(indent string, argument string) string {

	return addDoubleIndent(indent, "TYPE", argument)
}

func addTripleIndentOnlyFst(indent string, a1 string, a2 string, a3 string) string {
	var innerStats2 string = a3

	//var innerIndent string = fmt.Sprintf("%v%v", indent, basicIndent)

	var typeStats string = addIndAndNewLine(indent, a1)
	var innerStats string = a2
	if a3 != "" {
		innerStats2 = a3
	}

	return fmt.Sprintf("%v%v%v", typeStats, innerStats, innerStats2)
}

func addTripleIndent(indent string, a1 string, a2 string, a3 string) string {
	var innerStats2 string = a3

	var innerIndent string = fmt.Sprintf("%v%v", indent, basicIndent)

	var typeStats string = addIndAndNewLine(indent, a1)
	var innerStats string = addIndAndNewLine(innerIndent, a2)
	if a3 != "" {
		innerStats2 = addIndAndNewLine(innerIndent, a3)
	}

	return fmt.Sprintf("%v%v%v", typeStats, innerStats, innerStats2)
}

func (ast AST) ASTString() string {
	var tree string
	var tmpIndent string

	tree = addIndAndNewLine("", "Program")

	for _, function := range ast.functions {
		tree = fmt.Sprintf("%v%v", tree, function.ASTString(basicIndent))
	}

	tmpIndent = fmt.Sprintf("%v%v", basicIndent, basicIndent)

	tree = fmt.Sprintf("%v%v", tree, addIndAndNewLine(basicIndent, "int main()"))

	stmt := ast.main
	for stmt.GetNext() != nil {
		tree = fmt.Sprintf("%v\n%v ;", tree, stmt.ASTString(basicIndent))
		stmt = stmt.GetNext()
	}
	tree = fmt.Sprintf("%v%v", tree, stmt.ASTString(tmpIndent))

	return tree
}
