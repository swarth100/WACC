package main

// WACC Group 34
//
// ast.go: the structures for the AST the functions that parse the syntax tree
//
// Types, statements, expressions in the WACC language
// Functions to parse the WACC syntax tree into the AST

import (
	"errors"
	"fmt"
	"strconv"
)

// Type is an interface for WACC type
type Type interface {
	aststring(indent string) string
	Match(Type) bool
	String() string
}

// InvalidType is a WACC type for invalid constructs
type InvalidType struct{}

// UnknownType is a WACC type for cases where the type is not known
type UnknownType struct{}

// IntType is the WACC type for integers
type IntType struct{}

// BoolType is the WACC type for booleans
type BoolType struct{}

// CharType is the WACC type for characters
type CharType struct{}

// PairType is the WACC type for pairs
type PairType struct {
	first  Type
	second Type
}

// ArrayType is the WACC type for arrays
type ArrayType struct {
	base Type
}

// Expression is the interface for WACC expressions
type Expression interface {
	aststring(indent string) string
	TypeCheck(*Scope, chan<- error)
	GetType(*Scope) Type
	Token() *token32
	SetToken(*token32)
}

// Statement is the interface for WACC statements
type Statement interface {
	GetNext() Statement
	SetNext(Statement)
	istring(level int) string
	aststring(indent string) string
	TypeCheck(*Scope, chan<- error)
	Token() *token32
	SetToken(*token32)
}

// TokenBase is the base structure that contains the token reference
type TokenBase struct {
	token *token32
}

// Token returns the token in TokenBase
func (m *TokenBase) Token() *token32 {
	return m.token
}

// SetToken sets the current token in TokenBase
func (m *TokenBase) SetToken(token *token32) {
	m.token = token
}

// BaseStatement contains the pointer to the next statement
type BaseStatement struct {
	TokenBase
	next Statement
}

// GetNext returns the next statment in BaseStatment
func (m *BaseStatement) GetNext() Statement {
	return m.next
}

// SetNext sets the next statment in BaseStatment
func (m *BaseStatement) SetNext(next Statement) {
	m.next = next
}

// SkipStatement is the struct for WACC skip statement
type SkipStatement struct {
	BaseStatement
}

// BlockStatement is the struct for creating new block scope
type BlockStatement struct {
	BaseStatement
	body Statement
}

// DeclareAssignStatement declares a new variable and assigns the right hand
// side expression to it
type DeclareAssignStatement struct {
	BaseStatement
	waccType Type
	ident    string
	rhs      RHS
}

// LHS is the interface for the left hand side of an assignment
type LHS interface {
	aststring(indent string) string
	TypeCheck(*Scope, chan<- error)
	GetType(*Scope) Type
	Token() *token32
	SetToken(*token32)
}

// PairElemLHS is the struct for a pair on the lhs of an assignment
type PairElemLHS struct {
	TokenBase
	snd  bool
	expr Expression
}

// ArrayLHS is the struct for an array on the lhs of an assignment
type ArrayLHS struct {
	TokenBase
	ident string
	index []Expression
}

// VarLHS is the struct for a variable on the lhs of an assignment
type VarLHS struct {
	TokenBase
	ident string
}

// RHS is the interface for the right hand side of an assignment
type RHS interface {
	aststring(indent string) string
	TypeCheck(*Scope, chan<- error)
	GetType(*Scope) Type
	Token() *token32
	SetToken(*token32)
}

// PairLiterRHS is the struct for pair literals on the rhs of an assignment
type PairLiterRHS struct {
	TokenBase
	PairLiteral
}

// ArrayLiterRHS is the struct for array literals on the rhs of an assignment
type ArrayLiterRHS struct {
	TokenBase
	elements []Expression
}

// PairElemRHS is the struct for pair elements on the rhs of an assignment
type PairElemRHS struct {
	TokenBase
	snd  bool
	expr Expression
}

// FunctionCallRHS is the struct for function calls on the rhs of an assignment
type FunctionCallRHS struct {
	TokenBase
	ident string
	args  []Expression
}

// ExpressionRHS is the struct for expressions on the rhs of an assignment
type ExpressionRHS struct {
	TokenBase
	expr Expression
}

// AssignStatement is the struct for an assignment statement
type AssignStatement struct {
	BaseStatement
	target LHS
	rhs    RHS
}

// ReadStatement is the struct for a read statement
type ReadStatement struct {
	BaseStatement
	target LHS
}

// FreeStatement is the struct for a free statement
type FreeStatement struct {
	BaseStatement
	expr Expression
}

// ReturnStatement is the struct for a return statement
type ReturnStatement struct {
	BaseStatement
	expr Expression
}

// ExitStatement is the struct for an exit statement
type ExitStatement struct {
	BaseStatement
	expr Expression
}

// PrintLnStatement is the struct for a println statement
type PrintLnStatement struct {
	BaseStatement
	expr Expression
}

// PrintStatement is the struct for a print statement
type PrintStatement struct {
	BaseStatement
	expr Expression
}

// IfStatement is the struct for a if-else statement
type IfStatement struct {
	BaseStatement
	cond      Expression
	trueStat  Statement
	falseStat Statement
}

// WhileStatement is the struct for a while statement
type WhileStatement struct {
	BaseStatement
	cond Expression
	body Statement
}

// FunctionParam is the struct for a function parameter
type FunctionParam struct {
	TokenBase
	name     string
	waccType Type
}

// FunctionDef is the struct for a function definition
type FunctionDef struct {
	TokenBase
	ident      string
	returnType Type
	params     []*FunctionParam
	body       Statement
}

// AST is the main struct that represents the abstract syntax tree
type AST struct {
	main      Statement
	functions []*FunctionDef
}

// nodeRange given a node returns a channel from which all nodes at the same
// level can be read
func nodeRange(node *node32) <-chan *node32 {
	out := make(chan *node32)
	go func() {
		for ; node != nil; node = node.next {
			out <- node
		}
		close(out)
	}()
	return out
}

// nextNode given a node and a peg rule returns the first node in the chain
// that was created from that peg rule
func nextNode(node *node32, rule pegRule) *node32 {
	for cnode := range nodeRange(node) {
		if cnode.pegRule == rule {
			return cnode
		}
	}

	return nil
}

// parse array element access inside an expression
func parseArrayElem(node *node32) (Expression, error) {
	arrElem := &ArrayElem{}

	arrElem.ident = node.match

	// read and add all the indexer expressions
	for enode := nextNode(node, ruleEXPR); enode != nil; enode = nextNode(enode.next, ruleEXPR) {
		var exp Expression
		var err error
		if exp, err = parseExpr(enode.up); err != nil {
			return nil, err
		}
		arrElem.indexes = append(arrElem.indexes, exp)
	}

	return arrElem, nil
}

// Ident is the struct to represent an identifier
type Ident struct {
	TokenBase
	ident string
}

// IntLiteral is the struct to represent an integer literal
type IntLiteral struct {
	TokenBase
	value int
}

// BoolLiteralTrue is the struct to represent a true boolean literal
type BoolLiteralTrue struct {
	TokenBase
}

// BoolLiteralFalse is the struct to represent a false boolean literal
type BoolLiteralFalse struct {
	TokenBase
}

// CharLiteral is the struct to represent a character literal
type CharLiteral struct {
	TokenBase
	char string
}

// StringLiteral is the struct to represent a string literal
type StringLiteral struct {
	TokenBase
	str string
}

// PairLiteral is the struct to represent a pair literal
type PairLiteral struct {
	TokenBase
	fst Expression
	snd Expression
}

// NullPair is the struct to represent a null pair
type NullPair struct {
	TokenBase
}

// ArrayElem is the struct to represent an array element
type ArrayElem struct {
	TokenBase
	ident   string
	indexes []Expression
}

// UnaryOperator is the struct to represent the unary operators
type UnaryOperator interface {
	Expression
	GetExpression() Expression
	SetExpression(Expression)
}

// UnaryOperatorBase is the struct to represent the expression having the unary
// operator
type UnaryOperatorBase struct {
	TokenBase
	expr Expression
}

// GetExpression returns the expression associated with UnaryOperator
func (m *UnaryOperatorBase) GetExpression() Expression {
	return m.expr
}

// SetExpression sets the expression associated with UnaryOperator
func (m *UnaryOperatorBase) SetExpression(exp Expression) {
	m.expr = exp
}

// UnaryOperatorNot represents '!'
type UnaryOperatorNot struct {
	UnaryOperatorBase
}

// UnaryOperatorNegate represents '-'
type UnaryOperatorNegate struct {
	UnaryOperatorBase
}

// UnaryOperatorLen represents 'len'
type UnaryOperatorLen struct {
	UnaryOperatorBase
}

// UnaryOperatorOrd represents 'ord'
type UnaryOperatorOrd struct {
	UnaryOperatorBase
}

// UnaryOperatorChr represents 'chr'
type UnaryOperatorChr struct {
	UnaryOperatorBase
}

// BinaryOperator represents a generic binaryOperator which might be an expr.
type BinaryOperator interface {
	Expression
	GetRHS() Expression
	SetRHS(Expression)
	GetLHS() Expression
	SetLHS(Expression)
}

// BinaryOperator represents the base of a binary operator.
type BinaryOperatorBase struct {
	TokenBase
	lhs Expression
	rhs Expression
}

// GetLHS returns the left-hand-side associated with a BinaryOperatorBase.
func (m *BinaryOperatorBase) GetLHS() Expression {
	return m.lhs
}

// SetLHS sets the left-hand-side associated with a BinaryOperatorBase.
func (m *BinaryOperatorBase) SetLHS(exp Expression) {
	m.lhs = exp
}

// GetRHS returns the right-hand-side associated with a BinaryOperatorBase.
func (m *BinaryOperatorBase) GetRHS() Expression {
	return m.rhs
}

// SetRHS sets the right-hand-side associated with a BinaryOperatorBase.
func (m *BinaryOperatorBase) SetRHS(exp Expression) {
	m.rhs = exp
}

// BinaryOperatorMult represents '*'
type BinaryOperatorMult struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '/'
type BinaryOperatorDiv struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '%'
type BinaryOperatorMod struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '+'
type BinaryOperatorAdd struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '-'
type BinaryOperatorSub struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '>'
type BinaryOperatorGreaterThan struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '>='
type BinaryOperatorGreaterEqual struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '<'
type BinaryOperatorLessThan struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '<='
type BinaryOperatorLessEqual struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '=='
type BinaryOperatorEqual struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '!='
type BinaryOperatorNotEqual struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '&&'
type BinaryOperatorAnd struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '||'
type BinaryOperatorOr struct {
	BinaryOperatorBase
}

// BinaryOperatorMult represents '()'
type ExprParen struct {
	TokenBase
}

// exprStream given an expression node sends the all the nodes after it to
// channel skipping over spaces and flattening out the structure
func exprStream(node *node32) <-chan *node32 {
	out := make(chan *node32)
	go func() {
		for ; node != nil; node = node.next {
			switch node.pegRule {
			case ruleSPACE:
			case ruleBOOLLITER:
				out <- node.up
			case ruleEXPR:
				for inode := range exprStream(node.up) {
					out <- inode
				}
			default:
				out <- node
			}
		}
		close(out)
	}()
	return out
}

// parseExpr parses an expression and builds an expression tree that respects
// the operator precedence
// the function uses the shunting yard algorithm to achieve this
func parseExpr(node *node32) (Expression, error) {
	var stack []Expression
	var opstack []Expression

	// push an expression to the stack
	push := func(e Expression) {
		stack = append(stack, e)
	}

	// peek at the top of the expression stack
	peek := func() Expression {
		if len(stack) == 0 {
			return nil
		}
		return stack[len(stack)-1]
	}

	// pop and return the expression at the top the expression stack
	pop := func() (ret Expression) {
		ret, stack = stack[len(stack)-1], stack[:len(stack)-1]
		return
	}

	// push an operator to the operator stack
	pushop := func(e Expression) {
		opstack = append(opstack, e)
	}

	// peek at the top the operator stack
	peekop := func() Expression {
		if len(opstack) == 0 {
			return nil
		}
		return opstack[len(opstack)-1]
	}

	// pop and return the operator at the top of the operator stack
	popop := func() {
		var exp Expression

		exp, opstack = opstack[len(opstack)-1], opstack[:len(opstack)-1]

		switch t := exp.(type) {
		case UnaryOperator:
			t.SetExpression(pop())
		case BinaryOperator:
			t.SetRHS(pop())
			t.SetLHS(pop())
		case *ExprParen:
			exp = nil
		}

		if exp != nil {
			push(exp)
		}
	}

	// prio returns the priority of a given operator
	// the lesser the value the more tightly the operator binds
	// values taken from the operator precedence of C
	// special case parenthesis,  otherwise a high value
	prio := func(exp Expression) int {
		switch exp.(type) {
		case *UnaryOperatorNot:
			return 2
		case *UnaryOperatorNegate:
			return 2
		case *UnaryOperatorLen:
			return 2
		case *UnaryOperatorOrd:
			return 2
		case *UnaryOperatorChr:
			return 2
		case *BinaryOperatorMult:
			return 3
		case *BinaryOperatorDiv:
			return 3
		case *BinaryOperatorMod:
			return 3
		case *BinaryOperatorAdd:
			return 4
		case *BinaryOperatorSub:
			return 4
		case *BinaryOperatorGreaterThan:
			return 6
		case *BinaryOperatorGreaterEqual:
			return 6
		case *BinaryOperatorLessThan:
			return 6
		case *BinaryOperatorLessEqual:
			return 6
		case *BinaryOperatorEqual:
			return 7
		case *BinaryOperatorNotEqual:
			return 7
		case *BinaryOperatorAnd:
			return 11
		case *BinaryOperatorOr:
			return 12
		case *ExprParen:
			return 13
		default:
			return 42
		}
	}

	// returns whether the operator is right associative
	rightAssoc := func(exp Expression) bool {
		switch exp.(type) {
		case *UnaryOperatorNot:
			return true
		case *UnaryOperatorNegate:
			return true
		case *UnaryOperatorLen:
			return true
		case *UnaryOperatorOrd:
			return true
		case *UnaryOperatorChr:
			return true
		default:
			return false
		}
	}

	// given a peg rule return the operator with the expressions set
	ruleToOp := func(outer, inner pegRule) Expression {
		switch outer {
		case ruleUNARYOPER:
			switch inner {
			case ruleBANG:
				return &UnaryOperatorNot{}
			case ruleMINUS:
				return &UnaryOperatorNegate{}
			case ruleLEN:
				return &UnaryOperatorLen{}
			case ruleORD:
				return &UnaryOperatorOrd{}
			case ruleCHR:
				return &UnaryOperatorChr{}
			}
		case ruleBINARYOPER:
			switch inner {
			case ruleSTAR:
				return &BinaryOperatorMult{}
			case ruleDIV:
				return &BinaryOperatorDiv{}
			case ruleMOD:
				return &BinaryOperatorMod{}
			case rulePLUS:
				return &BinaryOperatorAdd{}
			case ruleMINUS:
				return &BinaryOperatorSub{}
			case ruleGT:
				return &BinaryOperatorGreaterThan{}
			case ruleGE:
				return &BinaryOperatorGreaterEqual{}
			case ruleLT:
				return &BinaryOperatorLessThan{}
			case ruleLE:
				return &BinaryOperatorLessEqual{}
			case ruleEQUEQU:
				return &BinaryOperatorEqual{}
			case ruleBANGEQU:
				return &BinaryOperatorNotEqual{}
			case ruleANDAND:
				return &BinaryOperatorAnd{}
			case ruleOROR:
				return &BinaryOperatorOr{}
			}
		}

		return nil
	}

	// process the nodes in order
	for enode := range exprStream(node) {
		switch enode.pegRule {
		case ruleINTLITER:
			num, err := strconv.ParseInt(enode.match, 10, 32)
			if err != nil {
				// number does not fit into WACC integer size
				numerr := err.(*strconv.NumError)
				switch numerr.Err {
				case strconv.ErrRange:
					return nil, CreateBigIntError(
						&enode.token32,
						enode.match,
					)
				}
				return nil, err
			}
			push(&IntLiteral{value: int(num)})
		case ruleFALSE:
			push(&BoolLiteralFalse{})
		case ruleTRUE:
			push(&BoolLiteralTrue{})
		case ruleCHARLITER:
			push(&CharLiteral{char: enode.up.next.match})
		case ruleSTRLITER:
			strLiter := &StringLiteral{}
			strNode := nextNode(enode.up, ruleSTR)
			if strNode != nil {
				// string may be empty, only set contents if not
				strLiter.str = strNode.match
			}
			push(strLiter)
		case rulePAIRLITER:
			push(&NullPair{})
		case ruleIDENT:
			push(&Ident{ident: enode.match})
		case ruleARRAYELEM:
			arrElem, err := parseArrayElem(enode.up)
			if err != nil {
				return nil, err
			}
			push(arrElem)
		case ruleUNARYOPER, ruleBINARYOPER:
			op1 := ruleToOp(enode.pegRule, enode.up.pegRule)
		op2l:
			for op2 := peekop(); op2 != nil; op2 = peekop() {
				if op2 == nil {
					break
				}

				// pop all operators with more tight binding
				switch {
				case !rightAssoc(op1) && prio(op1) >= prio(op2),
					rightAssoc(op1) && prio(op1) > prio(op2):
					popop()
				default:
					break op2l
				}
			}
			pushop(op1)
		case ruleLPAR:
			pushop(&ExprParen{})
		case ruleRPAR:
			// when a parenthesis is closed pop all the operators
			// the were inside
		parloop:
			for {
				switch peekop().(type) {
				case *ExprParen:
					popop()
					break parloop
				default:
					popop()
				}
			}
		}

		// set tokens on newly pushed expressions
		if val := peek(); val != nil && val.Token() == nil {
			peek().SetToken(&node.token32)
		}

		if op := peekop(); op != nil && op.Token() == nil {
			peekop().SetToken(&node.token32)
		}
	}

	// if operators are still left pop them
	for peekop() != nil {
		popop()
	}

	return pop(), nil
}

func parseLHS(node *node32) (LHS, error) {
	switch node.pegRule {
	case rulePAIRELEM:
		target := new(PairElemLHS)

		target.SetToken(&node.token32)

		fstNode := nextNode(node.up, ruleFST)
		target.snd = fstNode == nil

		exprNode := nextNode(node.up, ruleEXPR)
		var err error
		if target.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		return target, nil
	case ruleARRAYELEM:
		target := new(ArrayLHS)

		target.SetToken(&node.token32)

		identNode := nextNode(node.up, ruleIDENT)
		target.ident = identNode.match

		for exprNode := nextNode(node.up, ruleEXPR); exprNode != nil; exprNode = nextNode(exprNode.next, ruleEXPR) {
			var expr Expression
			var err error
			if expr, err = parseExpr(exprNode.up); err != nil {
				return nil, err
			}
			target.index = append(target.index, expr)
		}

		return target, nil
	case ruleIDENT:
		target := &VarLHS{ident: node.match}
		target.SetToken(&node.token32)
		return target, nil
	default:
		return nil, fmt.Errorf("Unexpected %s %s", node.String(), node.match)
	}
}

func parseRHS(node *node32) (RHS, error) {
	switch node.pegRule {
	case ruleNEWPAIR:
		var err error
		pair := new(PairLiterRHS)

		pair.SetToken(&node.token32)

		fstNode := nextNode(node, ruleEXPR)
		if pair.fst, err = parseExpr(fstNode.up); err != nil {
			return nil, err
		}

		sndNode := nextNode(fstNode.next, ruleEXPR)
		if pair.snd, err = parseExpr(sndNode.up); err != nil {
			return nil, err
		}

		return pair, nil
	case ruleARRAYLITER:
		node = node.up

		arr := new(ArrayLiterRHS)

		arr.SetToken(&node.token32)

		for node = nextNode(node, ruleEXPR); node != nil; node = nextNode(node.next, ruleEXPR) {
			var err error
			var expr Expression

			if expr, err = parseExpr(node.up); err != nil {
				return nil, err
			}
			arr.elements = append(arr.elements, expr)
		}

		return arr, nil
	case rulePAIRELEM:
		target := new(PairElemRHS)

		target.SetToken(&node.token32)

		fstNode := nextNode(node.up, ruleFST)
		target.snd = fstNode == nil

		exprNode := nextNode(node.up, ruleEXPR)
		var err error
		if target.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		return target, nil
	case ruleCALL:
		call := new(FunctionCallRHS)

		call.SetToken(&node.token32)

		identNode := nextNode(node, ruleIDENT)
		call.ident = identNode.match

		arglistNode := nextNode(node, ruleARGLIST)
		if arglistNode == nil {
			return call, nil
		}

		for argNode := nextNode(arglistNode.up, ruleEXPR); argNode != nil; argNode = nextNode(argNode.next, ruleEXPR) {
			var err error
			var expr Expression

			if expr, err = parseExpr(argNode.up); err != nil {
				return nil, err
			}

			call.args = append(call.args, expr)
		}

		return call, nil
	case ruleEXPR:
		exprRHS := new(ExpressionRHS)

		exprRHS.SetToken(&node.token32)

		var err error
		var expr Expression
		if expr, err = parseExpr(node.up); err != nil {
			return nil, err
		}

		exprRHS.expr = expr

		return exprRHS, nil
	default:
		return nil, fmt.Errorf("Unexpected rule %s %s", node.String(), node.match)
	}
}

func parseBaseType(node *node32) (Type, error) {
	switch node.pegRule {
	case ruleINT:
		return IntType{}, nil
	case ruleBOOL:
		return BoolType{}, nil
	case ruleCHAR:
		return CharType{}, nil
	case ruleSTRING:
		return ArrayType{base: CharType{}}, nil
	default:
		return nil, fmt.Errorf("Unknown type: %s", node.up.match)
	}
}

func parsePairType(node *node32) (Type, error) {
	var err error

	pairType := PairType{first: UnknownType{}, second: UnknownType{}}

	first := nextNode(node, rulePAIRELEMTYPE)

	second := nextNode(first.next, rulePAIRELEMTYPE)

	if pairType.first, err = parseType(first.up); err != nil {
		return nil, err
	}
	if pairType.second, err = parseType(second.up); err != nil {
		return nil, err
	}

	return pairType, nil

}

func parseType(node *node32) (Type, error) {
	var err error
	var waccType Type

	switch node.pegRule {
	case ruleBASETYPE:
		if waccType, err = parseBaseType(node.up); err != nil {
			return nil, err
		}
	case rulePAIRTYPE:
		if waccType, err = parsePairType(node.up); err != nil {
			return nil, err
		}
	case rulePAIR:
		return PairType{UnknownType{}, UnknownType{}}, nil
	}

	for node = nextNode(node.next, ruleARRAYTYPE); node != nil; node = nextNode(node.next, ruleARRAYTYPE) {
		waccType = ArrayType{base: waccType}
	}

	return waccType, nil
}

func parseStatement(node *node32) (Statement, error) {
	var stm Statement
	var err error

	switch node.pegRule {
	case ruleSKIP:
		stm = &SkipStatement{}
	case ruleBEGIN:
		block := new(BlockStatement)

		bodyNode := nextNode(node, ruleSTAT)
		if block.body, err = parseStatement(bodyNode.up); err != nil {
			return nil, err
		}

		stm = block
	case ruleTYPE:
		decl := new(DeclareAssignStatement)

		typeNode := nextNode(node, ruleTYPE)
		if decl.waccType, err = parseType(typeNode.up); err != nil {
			return nil, err
		}

		identNode := nextNode(node, ruleIDENT)
		decl.ident = identNode.match

		rhsNode := nextNode(node, ruleASSIGNRHS)
		if decl.rhs, err = parseRHS(rhsNode.up); err != nil {
			return nil, err
		}

		stm = decl
	case ruleASSIGNLHS:
		assign := new(AssignStatement)

		lhsNode := nextNode(node, ruleASSIGNLHS)
		if assign.target, err = parseLHS(lhsNode.up); err != nil {
			return nil, err
		}

		rhsNode := nextNode(node, ruleASSIGNRHS)
		if assign.rhs, err = parseRHS(rhsNode.up); err != nil {
			return nil, err
		}

		stm = assign
	case ruleREAD:
		read := new(ReadStatement)

		lhsNode := nextNode(node, ruleASSIGNLHS)
		if read.target, err = parseLHS(lhsNode.up); err != nil {
			return nil, err
		}

		stm = read
	case ruleFREE:
		free := new(FreeStatement)

		exprNode := nextNode(node, ruleEXPR)
		if free.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		stm = free
	case ruleRETURN:
		retur := new(ReturnStatement)

		exprNode := nextNode(node, ruleEXPR)
		if retur.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		stm = retur
	case ruleEXIT:
		exit := new(ExitStatement)

		exprNode := nextNode(node, ruleEXPR)
		if exit.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		stm = exit
	case rulePRINTLN:
		println := new(PrintLnStatement)

		exprNode := nextNode(node, ruleEXPR)
		if println.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		stm = println
	case rulePRINT:
		print := new(PrintStatement)

		exprNode := nextNode(node, ruleEXPR)
		if print.expr, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		stm = print
	case ruleIF:
		ifs := new(IfStatement)

		exprNode := nextNode(node, ruleEXPR)
		if ifs.cond, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		bodyNode := nextNode(node, ruleSTAT)
		if ifs.trueStat, err = parseStatement(bodyNode.up); err != nil {
			return nil, err
		}

		elseNode := nextNode(bodyNode.next, ruleSTAT)
		if ifs.falseStat, err = parseStatement(elseNode.up); err != nil {
			return nil, err
		}

		stm = ifs
	case ruleWHILE:
		whiles := new(WhileStatement)

		exprNode := nextNode(node, ruleEXPR)
		if whiles.cond, err = parseExpr(exprNode.up); err != nil {
			return nil, err
		}

		bodyNode := nextNode(node, ruleSTAT)
		if whiles.body, err = parseStatement(bodyNode.up); err != nil {
			return nil, err
		}

		stm = whiles
	default:
		return nil, fmt.Errorf(
			"unexpected %s %s",
			node.String(),
			node.match,
		)
	}

	if semi := nextNode(node, ruleSEMI); semi != nil {
		var next Statement
		if next, err = parseStatement(semi.next.up); err != nil {
			return nil, err
		}
		stm.SetNext(next)
	}

	stm.SetToken(&node.token32)

	return stm, nil
}

func parseParam(node *node32) (*FunctionParam, error) {
	var err error

	param := &FunctionParam{}

	param.SetToken(&node.token32)

	param.waccType, err = parseType(nextNode(node, ruleTYPE).up)
	if err != nil {
		return nil, err
	}

	param.name = nextNode(node, ruleIDENT).match

	return param, nil
}

func parseFunction(node *node32) (*FunctionDef, error) {
	var err error
	function := &FunctionDef{}

	function.SetToken(&node.token32)

	function.returnType, err = parseType(nextNode(node, ruleTYPE).up)
	if err != nil {
		return nil, err
	}

	function.ident = nextNode(node, ruleIDENT).match

	paramListNode := nextNode(node, rulePARAMLIST)
	if paramListNode != nil {
		for pnode := range nodeRange(paramListNode.up) {
			if pnode.pegRule == rulePARAM {
				var param *FunctionParam
				param, err = parseParam(pnode.up)
				if err != nil {
					return nil, err
				}
				function.params = append(function.params, param)
			}
		}
	}

	function.body, err = parseStatement(nextNode(node, ruleSTAT).up)
	if err != nil {
		return nil, err
	}

	return function, nil
}

func parseWACC(node *node32) (*AST, error) {
	ast := &AST{}

	for node := range nodeRange(node) {
		switch node.pegRule {
		case ruleBEGIN:
		case ruleEND:
		case ruleSPACE:
		case ruleFUNC:
			f, err := parseFunction(node.up)
			ast.functions = append(ast.functions, f)
			if err != nil {
				return nil, err
			}
		case ruleSTAT:
			var err error
			ast.main, err = parseStatement(node.up)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(
				"Unexpected %s %s",
				node.String(),
				node.match,
			)
		}
	}

	return ast, nil
}

// ParseAST given a syntax tree generated by the Peg library returns the
// internal representation of the WACC AST. On this AST further syntax and
// semantic analysis can be performed.
func ParseAST(wacc *WACC) (*AST, error) {
	node := wacc.AST()
	switch node.pegRule {
	case ruleWACC:
		return parseWACC(node.up)
	default:
		return nil, errors.New("expected ruleWACC")
	}
}
