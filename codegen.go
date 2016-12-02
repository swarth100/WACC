package main

// WACC Group 34
//
// codegen.go: Contains functions to codegen a given AST
//
// The File contains functions to codegen a given AST.

import (
	"bytes"
	"fmt"
	"sync"
)

//------------------------------------------------------------------------------
// RUNTIME ERRORS
//------------------------------------------------------------------------------

//Constant labels:
const (
	mPrintString          = "%.*s\\0"
	mPrintInt             = "%d\\0"
	mReadChar             = " %c\\0"
	mPrintReference       = "%p\\0"
	mNullChar             = "\\0"
	mNewLine              = "\\n\\0"
	mPutChar              = "putchar"
	mPuts                 = "puts"
	mTrue                 = "true\\0"
	mFalse                = "false\\0"
	mFFlush               = "fflush"
	mPrintf               = "printf"
	mScanf                = "scanf"
	mFreeLabel            = "free"
	mPrintNewLineLabel    = "p_print_ln"
	mPrintIntLabel        = "p_print_int"
	mPrintStringLabel     = "p_print_string"
	mPrintStringLoopLabel = "p_print_string_loop"
	mPrintStringEndLabel  = "p_print_string_return"
	mPrintCharLabel       = "p_print_char"
	mPrintBoolLabel       = "p_print_bool"
	mPrintReferenceLabel  = "p_print_reference"
	mReadIntLabel         = "p_read_int"
	mReadCharLabel        = "p_read_char"
	mExitLabel            = "exit"
	mMalloc               = "malloc"
	mThrowRuntimeErr      = "p_throw_runtime_error"
	mDivideByZeroLbl      = "p_check_divide_by_zero"
	mNullReferenceLbl     = "pi_check_null_pointer"
	mOverflowLbl          = "p_throw_overflow_error"
	mArrayBoundLbl        = "p_check_array_bounds"
	mDivideByZeroErr      = "DivideByZeroError: divide or modulo by zero\\n\\0"
	mNullReferenceErr     = "NullReferenceError: dereference a null reference" +
		"\\n\\0"
	mArrayNegIndexErr = "ArrayIndexOutOfBoundsError: negative index\\n\\0"
	mArrayLrgIndexErr = "ArrayIndexOutOfBoundsError: index too large\\n\\0"
	mOverflowErr      = "OverflowError: the result is too small/large to " +
		"store in a 4-byte signed-integer.\\n\\0"
)

//------------------------------------------------------------------------------
// INTERFACES
//------------------------------------------------------------------------------

// Instr is the interface for the ARM assembly instructions
type Instr interface {
	String() string
}

// Location is either a register or memory address
type Location interface {
	String() string
}

// Reg represents a register in ARM
type Reg interface {
	Location
	Reg() int
}

//------------------------------------------------------------------------------
// REG ALLOCATOR
//------------------------------------------------------------------------------

// ARMGenReg is a general purpose ARM register
type ARMGenReg struct {
	r int
}

func (m *ARMGenReg) String() string {
	return fmt.Sprintf("r%d", m.r)
}

// Reg returns the register number
func (m *ARMGenReg) Reg() int {
	return m.r
}

// ARMNamedReg is an ARM register with a specific purpose
type ARMNamedReg struct {
	r    int
	name string
}

func (m *ARMNamedReg) String() string {
	return m.name
}

// Reg returns the register number
func (m *ARMNamedReg) Reg() int {
	return m.r
}

// registers that can be used
var r0 = &ARMGenReg{r: 0}
var r1 = &ARMGenReg{r: 1}
var r2 = &ARMGenReg{r: 2}
var r3 = &ARMGenReg{r: 3}
var r4 = &ARMGenReg{r: 4}
var r5 = &ARMGenReg{r: 5}
var r6 = &ARMGenReg{r: 6}
var r7 = &ARMGenReg{r: 7}
var r8 = &ARMGenReg{r: 8}
var r9 = &ARMGenReg{r: 9}
var r10 = &ARMGenReg{r: 10}
var r11 = &ARMGenReg{r: 11}
var ip = &ARMNamedReg{name: "ip", r: 12}
var sp = &ARMNamedReg{name: "sp", r: 13}
var lr = &ARMNamedReg{name: "lr", r: 14}
var pc = &ARMNamedReg{name: "pc", r: 15}

var argRegs = []Reg{r0, r1, r2, r3}
var resReg = r0

// RegAllocator tracks register usage
type RegAllocator struct {
	usage        []int
	stringPool   *StringPool
	fsPool       *FSPool
	fname        string
	labelCounter int
	regs         []*ARMGenReg
	stackSize    int
	stack        []map[string]int
}

// CreateRegAllocator returns an allocator initialized with all the general
// purpose registers
func CreateRegAllocator() *RegAllocator {
	return &RegAllocator{
		regs: []*ARMGenReg{
			r4, r5, r6, r7, r8, r9, r10, r11,
		},
		usage: []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
}

// GetReg returns a register that is free and ready for use
func (m *RegAllocator) GetReg(insch chan<- Instr) Reg {
	r := m.regs[0]

	if m.usage[r.Reg()] > 0 {
		insch <- &PUSHInstr{
			BaseStackInstr: BaseStackInstr{
				regs: []Reg{r},
			},
		}
		m.PushStack(4)
	}

	m.usage[r.Reg()]++

	m.regs = append(m.regs[1:], r)

	return r
}

// FreeReg frees a register loading back the previous value if necessary
func (m *RegAllocator) FreeReg(re Reg, insch chan<- Instr) {
	if re.Reg() != m.regs[len(m.regs)-1].Reg() {
		panic("Register free order mismatch")
	}

	r := re.(*ARMGenReg)

	if m.usage[r.Reg()] > 1 {
		insch <- &POPInstr{
			BaseStackInstr: BaseStackInstr{
				regs: []Reg{r},
			},
		}
		m.PopStack(4)
	}

	m.usage[r.Reg()]--

	m.regs = append([]*ARMGenReg{r}, m.regs[:len(m.regs)-1]...)
}

// GetUniqueLabelSuffix returns a new unique label suffix
func (m *RegAllocator) GetUniqueLabelSuffix() string {
	defer func() {
		m.labelCounter++
	}()
	return fmt.Sprintf("_%s_%d", m.fname, m.labelCounter)
}

// PushStack increases the stack size by size
func (m *RegAllocator) PushStack(size int) {
	m.stackSize += size
}

// PopStack decreases the stack size by size
func (m *RegAllocator) PopStack(size int) {
	m.stackSize -= size
}

// DeclareVar registers a new variable for use
func (m *RegAllocator) DeclareVar(ident string, insch chan<- Instr) {
	m.PushStack(4)
	m.stack[0][ident] = m.stackSize
	insch <- &SUBInstr{
		BaseBinaryInstr: BaseBinaryInstr{
			dest: sp,
			lhs:  sp,
			rhs:  ImmediateOperand{4},
		},
	}
}

// ResolveVar returns the location of a variable
func (m *RegAllocator) ResolveVar(ident string) int {
	for _, scope := range m.stack {
		if v, ok := scope[ident]; ok {
			return (m.stackSize - v)
		}
	}

	panic(fmt.Sprintf("var %s not found in scope", ident))
}

// StartScope starts a new scope with new variable mappings possible
func (m *RegAllocator) StartScope(insch chan<- Instr) {
	m.stack = append([]map[string]int{make(map[string]int)}, m.stack...)
}

// CleanupScope starts a new scope with new variable mappings possible
func (m *RegAllocator) CleanupScope(insch chan<- Instr) {
	sl := len(m.stack[0]) * 4
	for o := sl; o > 0; o -= 255 {
		od := o
		if od > 255 {
			od = 255
		}
		insch <- &ADDInstr{
			BaseBinaryInstr: BaseBinaryInstr{
				dest: sp,
				lhs:  sp,
				rhs:  ImmediateOperand{od},
			},
		}
	}
	m.PopStack(sl)
	m.stack = m.stack[1:]
}

//-----------------------------------------------------------------------------
// GLOBAL SUPPORT FUNCTIONS
//-----------------------------------------------------------------------------

type FSPool struct {
	sync.RWMutex
	pool map[string]bool
}

func (m *FSPool) Add(function string) {
	m.Lock()
	defer m.Unlock()

	if m.pool == nil {
		m.pool = make(map[string]bool)
	}

	m.pool[function] = true
}

//------------------------------------------------------------------------------
// GLOBAL STRING STORAGE
//------------------------------------------------------------------------------

//DataString hold an int representing the length and a str field for the string
type DataString struct {
	len int
	str string
}

// StringPool holds the string literals that have been declared in the program
type StringPool struct {
	sync.RWMutex
	pool map[int]*DataString
}

// Lookup8 returns the msg label of a string literal
func (m *StringPool) Lookup8(msg string) string {
	m.Lock()
	defer m.Unlock()

	if m.pool == nil {
		m.pool = make(map[int]*DataString)
	}

	// TODO deduplicate strings

	l := len(m.pool)

	m.pool[l] = &DataString{len: len(msg), str: msg}

	return fmt.Sprintf("msg_%d", l)
}

// Lookup32 returns the msg label of a string literal, converted to 32 bit chars
// This allows for all values in WACC to be treated indifferently, as chars
// are converted to 32 bit chars
func (m *StringPool) Lookup32(msg string) string {
	m.Lock()
	defer m.Unlock()

	if m.pool == nil {
		m.pool = make(map[int]*DataString)
	}

	// TODO deduplicate strings

	l := len(m.pool)

	var buffer bytes.Buffer

	backslashCount := 0

	for i := 0; i < len(msg); i++ {
		if c := msg[i]; c == '\\' {
			backslashCount++
			buffer.WriteString(fmt.Sprintf("%c", msg[i]))
		} else {
			buffer.WriteString(fmt.Sprintf("%c\\000\\000\\000", msg[i]))
		}
	}

	m.pool[l] = &DataString{len: len(msg) - backslashCount, str: buffer.String()}

	return fmt.Sprintf("msg_%d", l)
}

//------------------------------------------------------------------------------
// CODEGEN
//------------------------------------------------------------------------------

// CodeGen base for next instruction
func (m *BaseStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	if m.next != nil {
		m.next.CodeGen(alloc, insch)
	}
}

// CodeGen for skip statements
// --> [CodeGen next instruction]
func (m *SkipStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen for block statements
// block_%l
// --> [CodeGen body]
// block_end_%l
// --> [CodeGen next instruction]
func (m *BlockStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	suffix := alloc.GetUniqueLabelSuffix()

	insch <- &LABELInstr{fmt.Sprintf("block%s", suffix)}
	alloc.StartScope(insch)

	m.body.CodeGen(alloc, insch)

	alloc.CleanupScope(insch)
	insch <- &LABELInstr{fmt.Sprintf("block_end%s", suffix)}

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for DeclareAssignStatement
// --> [CodeGen rhs] << reg
// --> STR reg [sp, #offset]
// --> [CodeGen next instruction]
func (m *DeclareAssignStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	lhs := m.ident
	alloc.DeclareVar(lhs, insch)

	rhs := m.rhs

	baseReg := alloc.GetReg(insch)
	rhs.CodeGen(alloc, baseReg, insch)

	storeValue := &MemoryStoreOperand{alloc.ResolveVar(lhs)}
	insch <- &STRInstr{StoreInstr{reg: baseReg, value: storeValue}}

	alloc.FreeReg(baseReg, insch)

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for AssignStatement
// --> [CodeGen lhs] << reg1
// --> [CodeGen rhs] << reg2
// --> STR reg2 [reg1]
// --> [CodeGen next instruction]
func (m *AssignStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	lhs := m.target

	rhs := m.rhs

	lhsReg := alloc.GetReg(insch)
	lhs.CodeGen(alloc, lhsReg, insch)

	rhsReg := alloc.GetReg(insch)
	rhs.CodeGen(alloc, rhsReg, insch)

	storeValue := &RegStoreOperand{lhsReg}
	insch <- &STRInstr{StoreInstr{reg: rhsReg, value: storeValue}}

	alloc.FreeReg(rhsReg, insch)
	alloc.FreeReg(lhsReg, insch)

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for ReadStatement
// --> [CodeGen target] << reg
// --> MOV r0, reg
// --> {int}: BL p_read_int
// --> {char}: BL p_read_char
// --> [CodeGen next instruction]
func (m *ReadStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	readReg := alloc.GetReg(insch)

	m.target.CodeGen(alloc, readReg, insch)

	insch <- &MOVInstr{dest: r0, source: readReg}

	switch m.target.Type().(type) {
	case IntType:
		alloc.fsPool.Add(mReadIntLabel)
		insch <- &BLInstr{
			BInstr: BInstr{label: mReadIntLabel},
		}
	case CharType:
		alloc.fsPool.Add(mReadCharLabel)
		insch <- &BLInstr{BInstr: BInstr{label: mReadCharLabel}}
	default:
		panic(fmt.Errorf("%v has no type information", m.target))
	}

	alloc.FreeReg(readReg, insch)
	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for FreeStatement
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> BL pi_check_null_pointer
// --> MOV r0, reg
// --> BL free
// --> [CodeGen next instruction]
func (m *FreeStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	reg := alloc.GetReg(insch)

	m.expr.CodeGen(alloc, reg, insch)

	insch <- &MOVInstr{dest: r0, source: reg}

	insch <- &BLInstr{BInstr{label: mNullReferenceLbl}}

	insch <- &MOVInstr{dest: r0, source: reg}

	insch <- &BLInstr{BInstr{label: mFreeLabel}}

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for ReturnStatement
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> ADD sp, sp, #offset
// --> B %l_return
// --> [CodeGen next instruction]
func (m *ReturnStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	reg := alloc.GetReg(insch)

	m.expr.CodeGen(alloc, reg, insch)
	insch <- &MOVInstr{dest: resReg, source: reg}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: sp, lhs: sp,
		rhs: ImmediateOperand{alloc.stackSize}}}

	insch <- &BInstr{label: fmt.Sprintf("%s_return", alloc.fname)}

	alloc.FreeReg(reg, insch)

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for ExitStatement
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> BL exit
// --> [CodeGen next instruction]
func (m *ExitStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	reg := alloc.GetReg(insch)

	m.expr.CodeGen(alloc, reg, insch)

	insch <- &MOVInstr{dest: r0, source: reg}

	insch <- &BLInstr{BInstr: BInstr{label: "exit"}}

	alloc.FreeReg(reg, insch)

	m.BaseStatement.CodeGen(alloc, insch)
}

func print(m Expression, alloc *RegAllocator, insch chan<- Instr) {
	r := alloc.GetReg(insch)
	m.CodeGen(alloc, r, insch)
	insch <- &MOVInstr{dest: r0, source: r}
	alloc.FreeReg(r, insch)
	switch t := m.Type().(type) {
	case IntType:
		alloc.fsPool.Add(mPrintIntLabel)
		insch <- &BLInstr{BInstr: BInstr{label: mPrintIntLabel}}
	case BoolType:
		alloc.fsPool.Add(mPrintBoolLabel)
		insch <- &BLInstr{BInstr: BInstr{label: mPrintBoolLabel}}
	case CharType:
		alloc.fsPool.Add(mPrintCharLabel)
		insch <- &BLInstr{BInstr: BInstr{label: mPrintCharLabel}}
	case PairType:
		alloc.fsPool.Add(mPrintReferenceLabel)
		insch <- &BLInstr{BInstr: BInstr{label: mPrintReferenceLabel}}
	case ArrayType:
		switch t.base.(type) {
		case CharType:
			alloc.fsPool.Add(mPrintStringLabel)
			insch <- &BLInstr{BInstr: BInstr{label: mPrintStringLabel}}
		default:
			alloc.fsPool.Add(mPrintReferenceLabel)
			insch <- &BLInstr{BInstr: BInstr{label: mPrintReferenceLabel}}
		}
	default:
		panic(fmt.Errorf("%v has no type information", m))
	}
}

//CodeGen generates code for PrintLnStatement
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> BL {depends on type}
// --> BL p_print_ln
// --> [CodeGen next instruction]
func (m *PrintLnStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	print(m.expr, alloc, insch)

	insch <- &BLInstr{BInstr{label: mPrintNewLineLabel}}

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for PrintStatement
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> BL {depends on type}
// --> [CodeGen next instruction]
func (m *PrintStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	print(m.expr, alloc, insch)
	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for IfStatement
// if_%l
// --> [CodeGen condition] << reg
// --> CMP reg, 0
// --> BEQ else_%l
// then_%l
// --> [CodeGen trueStat]
// --> B end_%l
// else_%l
// --> [CodeGen falseStat]
// end_%l
// --> [CodeGen next instruction]
func (m *IfStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	suffix := alloc.GetUniqueLabelSuffix()

	labelIf := fmt.Sprintf("if%s", suffix)
	labelThen := fmt.Sprintf("then%s", suffix)
	labelElse := fmt.Sprintf("else%s", suffix)
	labelEnd := fmt.Sprintf("end%s", suffix)

	// Condition
	insch <- &LABELInstr{ident: labelIf}
	target := alloc.GetReg(insch)

	m.cond.CodeGen(alloc, target, insch)

	// CMP Check
	TruthValue := &ImmediateOperand{0}
	insch <- &CMPInstr{BaseComparisonInstr{lhs: target, rhs: TruthValue}}

	alloc.FreeReg(target, insch)

	insch <- &BInstr{label: labelElse, cond: condEQ}

	//TruthCases
	insch <- &LABELInstr{ident: labelThen}
	alloc.StartScope(insch)

	m.trueStat.CodeGen(alloc, insch)

	alloc.CleanupScope(insch)
	insch <- &BInstr{label: labelEnd}

	//FalseCases
	insch <- &LABELInstr{ident: labelElse}
	alloc.StartScope(insch)

	m.falseStat.CodeGen(alloc, insch)

	alloc.CleanupScope(insch)
	insch <- &LABELInstr{ident: labelEnd}

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for WhileStatement
// while_%l
// --> B cond_%l
// do_%l
// --> [CodeGen body]
// cond_%l
// --> [CodeGen cond] << reg
// --> CMP reg, 1
// --> BEQ do_%l
// end_%l
// --> [CodeGen next instruction]
func (m *WhileStatement) CodeGen(alloc *RegAllocator, insch chan<- Instr) {
	suffix := alloc.GetUniqueLabelSuffix()

	labelWhile := fmt.Sprintf("while%s", suffix)
	labelCond := fmt.Sprintf("cond%s", suffix)
	labelDo := fmt.Sprintf("do%s", suffix)
	labelEnd := fmt.Sprintf("end%s", suffix)

	// CMP Check

	insch <- &LABELInstr{ident: labelWhile}
	insch <- &BInstr{label: labelCond}

	//Body
	insch <- &LABELInstr{ident: labelDo}
	alloc.StartScope(insch)

	m.body.CodeGen(alloc, insch)

	alloc.CleanupScope(insch)

	// Condition
	insch <- &LABELInstr{ident: labelCond}
	target := alloc.GetReg(insch)

	m.cond.CodeGen(alloc, target, insch)

	alloc.FreeReg(target, insch)

	insch <- &CMPInstr{BaseComparisonInstr{lhs: target, rhs: &ImmediateOperand{1}}}

	insch <- &BInstr{label: labelDo, cond: condEQ}

	insch <- &LABELInstr{ident: labelEnd}

	m.BaseStatement.CodeGen(alloc, insch)
}

//CodeGen generates code for PairElemLHS
// --> [CodeGen expr] << reg
// --> MOV r0, reg
// --> BL p_print_ln
func (m *PairElemLHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	pairElem(m.expr, alloc, target, insch)

	if m.snd {
		insch <- &ADDInstr{BaseBinaryInstr{dest: target, lhs: target, rhs: &ImmediateOperand{4}}}
	}
}

func arrayHelper(ident string, exprs []Expression, alloc *RegAllocator, target Reg, insch chan<- Instr) {
	//Load array Address
	rhsVal := &ImmediateOperand{alloc.ResolveVar(ident)}
	insch <- &ADDInstr{BaseBinaryInstr{dest: target, lhs: sp, rhs: rhsVal}}

	//Place index in new Register
	indexReg := alloc.GetReg(insch)
	for index := 0; index < len(exprs); index++ {

		//Retrieve content of Array Address
		insch <- &LDRInstr{LoadInstr{reg: target, value: &RegisterLoadOperand{reg: target}}}

		exprs[index].CodeGen(alloc, indexReg, insch)

		//Check array Bounds
		insch <- &MOVInstr{dest: r0, source: indexReg}
		insch <- &MOVInstr{dest: r1, source: target}
		insch <- &BLInstr{BInstr{label: mArrayBoundLbl}}

		//Target now points to the first element
		rhsVal := &ImmediateOperand{4}
		insch <- &ADDInstr{BaseBinaryInstr{dest: target, lhs: target, rhs: rhsVal}}

		//Target now points to the index element
		OpTwoRegLSL := &LSLRegOperand{reg: indexReg, offset: 2}
		insch <- &ADDInstr{BaseBinaryInstr{dest: target, lhs: target, rhs: OpTwoRegLSL}}
	}

	alloc.FreeReg(indexReg, insch)
}

//CodeGen generates code for ArrayLHS
// --> ADD target, sp, #offset
// --> LDR target, [target]
// --> [Codegen index] << reg
// --> MOV r0, reg
// --> MOV r1, target
// --> BL p_check_array_bounds
// --> ADD target, target, #4
// --> ADD target, target, [reg, LSL 2]
func (m *ArrayLHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	arrayHelper(m.ident, m.index, alloc, target, insch)
}

//CodeGen generates code for VarLHS
// --> MOV target, sp
// --> ADD target, target, #offset
func (m *VarLHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &MOVInstr{dest: target, source: sp}
	rhsVal := &ImmediateOperand{alloc.ResolveVar(m.ident)}
	insch <- &ADDInstr{BaseBinaryInstr{dest: target, lhs: target, rhs: rhsVal}}
}

//CodeGen generates code for PairLiterRHS
// --> [CodeGen PairLiteral]
func (m *PairLiterRHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.PairLiteral.CodeGen(alloc, target, insch)
}

//CodeGen generates code for ArrayLiterRHS
// --> LDR r0, =(length+1)*4
// --> BL malloc
// --> MOV target, r0
// --> [Codegen elem] << reg
// --> STR reg, [target, #offset]
// --> LDR reg, #length
// --> STR reg, [target]
func (m *ArrayLiterRHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {

	//Call Malloc
	leng := &ConstLoadOperand{len(m.elements)*4 + 4}
	insch <- &LDRInstr{LoadInstr{reg: r0, value: leng}}

	insch <- &BLInstr{BInstr{label: mMalloc}}

	insch <- &MOVInstr{dest: target, source: resReg}

	//Array Pos Reg
	arrayReg := alloc.GetReg(insch)

	//Populate Heap at array indexes
	for pos := 1; pos <= len(m.elements); pos++ {
		element := m.elements[pos-1]
		element.CodeGen(alloc, arrayReg, insch)

		regOffset := &RegStoreOffsetOperand{reg: target, offset: (pos * 4)}
		insch <- &STRInstr{StoreInstr{reg: arrayReg, value: regOffset}}
	}

	alloc.FreeReg(arrayReg, insch)

	//Mov length into position 0
	lenInt := &ConstLoadOperand{len(m.elements)}
	insch <- &LDRInstr{LoadInstr{reg: arrayReg, value: lenInt}}

	insch <- &STRInstr{StoreInstr{reg: arrayReg, value: &RegStoreOperand{target}}}
}

func pairElem(expr Expression, alloc *RegAllocator, target Reg, insch chan<- Instr) {
	expr.CodeGen(alloc, target, insch)

	//Mov + CheckNullPointer Label
	insch <- &MOVInstr{dest: r0, source: target}
	insch <- &BLInstr{BInstr{label: mNullReferenceLbl}}
}

//CodeGen generates code for PairElemRHS
// --> MOV r0, target
// --> BL pi_check_null_pointer
// --> LDR target, [target, #offset]
func (m *PairElemRHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	pairElem(m.expr, alloc, target, insch)

	offset := 0

	if m.snd {
		offset = 4
	}

	//Load fst or snd
	insch <- &LDRInstr{LoadInstr{reg: target, value: &RegisterLoadOperand{reg: target, value: offset}}}
}

//CodeGen generates code for FunctionCallRHS
// [CodeGen param] << reg
// PUSH reg
// POP r0,r1,r2,r3
// BL f
// MOV target, r0
// POP [params]
func (m *FunctionCallRHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	for i := len(m.args) - 1; i >= 0; i-- {
		reg := alloc.GetReg(insch)
		m.args[i].CodeGen(alloc, reg, insch)
		insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{reg}}}
		alloc.PushStack(4)
		alloc.FreeReg(reg, insch)
	}

	for i := 0; i < 4 && i < len(m.args); i++ {
		insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{argRegs[i]}}}
	}

	insch <- &BLInstr{BInstr: BInstr{label: m.ident}}

	insch <- &MOVInstr{dest: target, source: resReg}

	if pl := len(m.args); pl > 4 {
		insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: sp, lhs: sp,
			rhs: ImmediateOperand{(pl - 4) * 4}}}
	}

	alloc.PopStack(len(m.args) * 4)
}

//CodeGen generates code for ExpressionRHS
// --> [Codegen expr]
func (m *ExpressionRHS) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)
}

//------------------------------------------------------------------------------
// LITERALS AND ELEMENTS CODEGEN
//------------------------------------------------------------------------------

//CodeGen generates code for Ident
// --> LDR target, [sp, #offset]
func (m *Ident) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	loadValue := &RegisterLoadOperand{reg: sp, value: alloc.ResolveVar(m.ident)}
	insch <- &LDRInstr{LoadInstr{reg: target, value: loadValue}}
}

//CodeGen generates code for IntLiteral
// --> LDR target, =offset
func (m *IntLiteral) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	loadValue := &ConstLoadOperand{m.value}
	insch <- &LDRInstr{LoadInstr{reg: target, value: loadValue}}
}

//CodeGen generates code for BoolLiteralTrue
// --> MOV target, 1
func (m *BoolLiteralTrue) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &MOVInstr{dest: target, source: &ImmediateOperand{1}}
}

//CodeGen generates code for BoolLiteralFalse
// --> MOV target, 0
func (m *BoolLiteralFalse) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &MOVInstr{dest: target, source: &ImmediateOperand{0}}
}

//CodeGen generates code for CharLiteral
// --> MOV target, #char
func (m *CharLiteral) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &MOVInstr{dest: target, source: CharOperand{m.char}}
}

//CodeGen generates code for StringLiteral
// --> LDR target, =msg_x
func (m *StringLiteral) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup32(m.str)

	insch <- &LDRInstr{
		LoadInstr: LoadInstr{reg: target, value: &BasicLoadOperand{msg}}}
}

//CodeGen generates code for PairLiteral
// --> LDR r0, =8
// --> BL malloc
// --> MOV target, r0
// --> [Codegen fst] << reg
// --> STR reg, [target]
// --> [Codegen snd] << reg
// --> STR reg, [target, #4]
func (m *PairLiteral) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &LDRInstr{LoadInstr{reg: r0, value: &ConstLoadOperand{8}}}
	insch <- &BLInstr{BInstr{label: mMalloc}}
	//target cointains address of newpair
	insch <- &MOVInstr{dest: target, source: resReg}
	elemReg := alloc.GetReg(insch)
	m.fst.CodeGen(alloc, elemReg, insch)
	insch <- &STRInstr{StoreInstr{reg: elemReg, value: &RegStoreOperand{target}}}
	m.snd.CodeGen(alloc, elemReg, insch)
	insch <- &STRInstr{StoreInstr{reg: elemReg,
		value: &RegStoreOffsetOperand{reg: target, offset: 4}}}
	alloc.FreeReg(elemReg, insch)
}

//CodeGen generates code for NullPair
// --> MOV target, r0
func (m *NullPair) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	insch <- &MOVInstr{dest: target, source: ImmediateOperand{0}}
}

//CodeGen generates code for ArrayElem
// --> ADD target, sp, #offset
// --> LDR target, [target]
// --> [Codegen index] << reg
// --> MOV r0, reg
// --> MOV r1, target
// --> BL p_check_array_bounds
// --> ADD target, target, #4
// --> ADD target, target, [reg, LSL 2]
// --> LDR target, [target]
func (m *ArrayElem) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	arrayHelper(m.ident, m.indexes, alloc, target, insch)

	insch <- &LDRInstr{LoadInstr{reg: target, value: &RegisterLoadOperand{reg: target}}}
}

//------------------------------------------------------------------------------
// UNARY OPERATOR CODEGEN
//------------------------------------------------------------------------------

//CodeGen generates code for UnaryOperatorNot
// --> EOR target, target, #1
func (m *UnaryOperatorNot) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)
	insch <- &NOTInstr{BaseUnaryInstr{arg: target, dest: target}}
}

//CodeGen generates code for UnaryOperatorNegate
// --> NEGS target, target
// --> BL p_throw_overflow_error
func (m *UnaryOperatorNegate) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)
	insch <- &NEGInstr{BaseUnaryInstr{arg: target, dest: target}}

	insch <- &BLInstr{BInstr: BInstr{cond: condVS, label: mOverflowLbl}}
}

//CodeGen generates code for UnaryOperatorLen
// --> [CodeGen expr]
// --> LDR target, [target]
func (m *UnaryOperatorLen) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)

	//Load length into target
	insch <- &LDRInstr{LoadInstr{reg: target, value: &RegisterLoadOperand{reg: target}}}
}

//CodeGen generates code for UnaryOperatorOrd
// --> [CodeGen expr]
func (m *UnaryOperatorOrd) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)
}

//CodeGen generates code for UnaryOperatorChr
// --> [CodeGen expr]
func (m *UnaryOperatorChr) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	m.expr.CodeGen(alloc, target, insch)
}

//------------------------------------------------------------------------------
// BINARY OPERATOR CODEGEN
//------------------------------------------------------------------------------

//CodeGen generates code for BinaryOperatorMult
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
//CodeGen generates code for BinaryOperatorMult
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> SMULL target, target2, target, target2
// --> CMP target2, target, ASR #31
// --> BLNE p_throw_overflow_error
func (m *BinaryOperatorMult) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
	}
	binaryInstrMul := &SMULLInstr{RdLo: target, RdHi: target2, Rm: target,
		Rs: target2}

	alloc.fsPool.Add(mOverflowLbl)

	alloc.FreeReg(target2, insch)
	insch <- binaryInstrMul

	insch <- &CMPInstr{BaseComparisonInstr: BaseComparisonInstr{lhs: target2,
		rhs: &RegisterOperand{reg: target, shift: shiftASR, amount: 31}}}

	insch <- &BLInstr{BInstr: BInstr{cond: condNE, label: mOverflowLbl}}
}

//CodeGen generates code for BinaryOperatorDiv
//CodeGen generates code for BinaryOperatorDiv
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> MOV r0, lhsResult
// --> MOV r1, rhsResult
// --> BL p_check_divide_by_zero
// --> BL __aeabi_idiv
// --> MOV target, r0
func (m *BinaryOperatorDiv) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	var rhsResult Reg
	lhsResult := target
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
		rhsResult = target2
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
		lhsResult = target2
		rhsResult = target
	}

	alloc.fsPool.Add(mDivideByZeroLbl)

	insch <- &MOVInstr{dest: r0, source: lhsResult}
	insch <- &MOVInstr{dest: r1, source: rhsResult}
	insch <- &BLInstr{BInstr: BInstr{label: mDivideByZeroLbl}}
	insch <- &BLInstr{BInstr: BInstr{label: "__aeabi_idiv"}}
	insch <- &MOVInstr{dest: target, source: resReg}
	alloc.FreeReg(target2, insch)

}

//CodeGen generates code for BinaryOperatorMod
//CodeGen generates code for BinaryOperatorMod
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> MOV r0, lhsResult
// --> MOV r1, rhsResult
// --> BL p_check_divide_by_zero
// --> BL __aeabi_idivmod
// --> MOV target, r1
func (m *BinaryOperatorMod) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	var rhsResult Reg
	lhsResult := target
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
		rhsResult = target2
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
		lhsResult = target2
		rhsResult = target
	}

	alloc.fsPool.Add(mDivideByZeroLbl)

	insch <- &MOVInstr{dest: r0, source: lhsResult}
	insch <- &MOVInstr{dest: r1, source: rhsResult}
	insch <- &BLInstr{BInstr: BInstr{label: mDivideByZeroLbl}}
	insch <- &BLInstr{BInstr: BInstr{label: "__aeabi_idivmod"}}
	insch <- &MOVInstr{dest: target, source: r1}
	alloc.FreeReg(target2, insch)
}

//CodeGen generates code for BinaryOperatorAdd
//CodeGen generates code for BinaryOperatorAdd
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> ADD target, target2, target
// --> MOV r1, rhsResult
// --> BLVS p_throw_overflow_error
func (m *BinaryOperatorAdd) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
	}

	alloc.fsPool.Add(mOverflowLbl)

	binaryInstrAdd := &ADDInstr{BaseBinaryInstr{dest: target, lhs: target2,
		rhs: target}}
	alloc.FreeReg(target2, insch)
	insch <- binaryInstrAdd

	insch <- &BLInstr{
		BInstr: BInstr{
			cond:  condVS,
			label: mOverflowLbl,
		},
	}
}

//CodeGen generates code for BinaryOperatorSub
//CodeGen generates code for BinaryOperatorSub
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> SUB target, target, target2
// --> BLVS p_throw_overflow_errorcode
func (m *BinaryOperatorSub) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	var binaryInstrSub *SUBInstr
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
		binaryInstrSub = &SUBInstr{BaseBinaryInstr{dest: target,
			lhs: target, rhs: target2}}
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
		binaryInstrSub = &SUBInstr{BaseBinaryInstr{dest: target,
			lhs: target2, rhs: target}}
	}

	alloc.fsPool.Add(mOverflowLbl)

	alloc.FreeReg(target2, insch)
	insch <- binaryInstrSub

	insch <- &BLInstr{BInstr: BInstr{cond: condVS, label: mOverflowLbl}}
}

//CodeGenComparators is a helper function for CodeGen over Comparator instructions
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> CMP target2, target
// --> MOV(COND) target, 1
// --> MOV(NOT-COND) target, 0
func codeGenComparators(m BinaryOperator, alloc *RegAllocator, target Reg, insch chan<- Instr, condCode int) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	var binaryInstrCMP *CMPInstr
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
		binaryInstrCMP = &CMPInstr{BaseComparisonInstr{lhs: target2,
			rhs: target}}
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
		binaryInstrCMP = &CMPInstr{BaseComparisonInstr{lhs: target2,
			rhs: target}}
	}
	alloc.FreeReg(target2, insch)
	insch <- binaryInstrCMP
	insch <- &MOVInstr{cond: Cond(condCode), dest: target,
		source: ImmediateOperand{1}}
	insch <- &MOVInstr{cond: Cond(condCode).getOpposite(), dest: target,
		source: ImmediateOperand{0}}
}

//CodeGen generates code for BinaryOperatorGreaterThan
//Calls codeGenComparators helper function
func (m *BinaryOperatorGreaterThan) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condGT)
}

//CodeGen generates code for BinaryOperatorGreaterEqual
//Calls codeGenComparators helper function
func (m *BinaryOperatorGreaterEqual) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condGE)
}

//CodeGen generates code for BinaryOperatorLessThan
//Calls codeGenComparators helper function
func (m *BinaryOperatorLessThan) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condLT)
}

//CodeGen generates code for BinaryOperatorLessEqual
//Calls codeGenComparators helper function
func (m *BinaryOperatorLessEqual) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condLE)
}

//CodeGen generates code for BinaryOperatorEqual
//Calls codeGenComparators helper function
func (m *BinaryOperatorEqual) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condEQ)
}

//CodeGen generates code for BinaryOperatorNotEqual
//Calls codeGenComparators helper function
func (m *BinaryOperatorNotEqual) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	codeGenComparators(m, alloc, target, insch, condNE)
}

//CodeGen generates code for BinaryOperatorAnd
//CodeGen generates code for BinaryOperatorAnd
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> AND target, target2, target
func (m *BinaryOperatorAnd) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
	}
	binaryInstrAnd := &ANDInstr{BaseBinaryInstr{dest: target, lhs: target2,
		rhs: target}}
	alloc.FreeReg(target2, insch)
	insch <- binaryInstrAnd
}

//CodeGen generates code for BinaryOperatorOr
//CodeGen generates code for BinaryOperatorOr
//CodeGen generates code for BinaryOperatorAnd
// If LHS.Weight > RHS.Weight LHS is executed first
// otherwise RHS is executed first
// --> [CodeGen exprLHS] < target
// --> [CodeGen exprRHS] < target2
// --> ORR target, target2, target
func (m *BinaryOperatorOr) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
	lhs := m.GetLHS()
	rhs := m.GetRHS()
	var target2 Reg
	if lhs.Weight() > rhs.Weight() {
		lhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		rhs.CodeGen(alloc, target2, insch)
	} else {
		rhs.CodeGen(alloc, target, insch)
		target2 = alloc.GetReg(insch)
		lhs.CodeGen(alloc, target2, insch)
	}
	binaryInstrOrr := &ORRInstr{BaseBinaryInstr{dest: target, lhs: target2,
		rhs: target}}
	alloc.FreeReg(target2, insch)
	insch <- binaryInstrOrr
}

//CodeGen generates code for ExprParen
func (m *ExprParen) CodeGen(alloc *RegAllocator, target Reg, insch chan<- Instr) {
}

//------------------------------------------------------------------------------
// WEIGHT FUNCTIONS
//------------------------------------------------------------------------------

//Weight returns weight of Ident
func (m *Ident) Weight() int {
	return 1
}

//Weight returns weight of IntLiteral
func (m *IntLiteral) Weight() int {
	return 1
}

//Weight returns weight of BoolLiteralTrue
func (m *BoolLiteralTrue) Weight() int {
	return 1
}

//Weight returns weight of BoolLiteralFalse
func (m *BoolLiteralFalse) Weight() int {
	return 1
}

//Weight returns weight of CharLiteral
func (m *CharLiteral) Weight() int {
	return 1
}

//Weight returns weight of StringLiteral
func (m *StringLiteral) Weight() int {
	return 1
}

//Weight returns weight of PairLiteral
func (m *PairLiteral) Weight() int {
	if m.weightCache > 0 {
		return m.weightCache
	}
	m.weightCache = maxWeight(m.fst.Weight(), m.snd.Weight()) + 1
	return m.weightCache
}

//Weight returns weight of NullPair
func (m *NullPair) Weight() int {
	return 1
}

//Weight returns weight of ArrayElem
func (m *ArrayElem) Weight() int {
	if m.weightCache > 0 {
		return m.weightCache
	}
	for _, index := range m.indexes {
		iw := index.Weight()
		if m.weightCache > iw {
			m.weightCache = iw
		}
	}
	m.weightCache++
	return m.weightCache
}

//Weight returns weight of all UnaryOperators
func (m *UnaryOperatorBase) Weight() int {
	if m.weightCache > 0 {
		return m.weightCache
	}
	m.weightCache = m.expr.Weight()
	return m.weightCache
}

func maxWeight(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func minWeight(x, y int) int {
	if x < y {
		return x
	}
	return y
}

//Weight returns weight of all BinaryOperators
func (m *BinaryOperatorBase) Weight() int {
	if m.weightCache > 0 {
		return m.weightCache
	}
	cost1 := maxWeight(m.lhs.Weight(), m.rhs.Weight()+1)
	cost2 := maxWeight(m.lhs.Weight()+1, m.rhs.Weight())
	m.weightCache = minWeight(cost1, cost2)
	return m.weightCache
}

//Weight returns weight of ExprParen
func (m *ExprParen) Weight() int {
	return -1
}

//------------------------------------------------------------------------------
// ASSEMBLY UTIL FUNCTIONS
//------------------------------------------------------------------------------

//printNewLine generates code to print a New Line
// p_print_ln:
// --> 	PUSH {lr}
// -->	LDR r0, =msg_4
// -->	ADDS r0, r0, #4
// -->	BL printf
// -->	MOV r0, #0
// -->	BL fflush
// -->	POP {pc}
func printNewLine(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mNewLine)

	insch <- &LABELInstr{mPrintNewLineLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &LDRInstr{LoadInstr{reg: r0, value: &BasicLoadOperand{msg}}}

	insch <- &ADDInstr{BaseBinaryInstr{dest: r0, lhs: r0, rhs: ImmediateOperand{4}}}

	insch <- &BLInstr{BInstr{label: mPrintf}}

	insch <- &MOVInstr{dest: r0, source: ImmediateOperand{0}}

	insch <- &BLInstr{BInstr{label: mFFlush}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//printString generates code to print a given string
// p_print_string:
// -->	PUSH {lr}
// -->	PUSH {r4, r5}
// -->	LDR r4, [r0]
// -->	ADDS r5, r0, #4
// p_print_string_loop:
// -->	TEQ r4, #0
// -->	BEQ p_print_string_return
// -->	LDR r0, [r5]
// -->	BL putchar
// -->	SUBS r4, r4, #1
// -->	ADDS r5, r5, #4
// -->	B p_print_string_loop
// p_print_string_return:
// -->	MOV r0, #0
// -->	BL fflush
// -->	POP {r4, r5}
// -->	POP {pc}
func printString(alloc *RegAllocator, insch chan<- Instr) {
	insch <- &LABELInstr{mPrintStringLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{r4, r5}}}

	insch <- &LDRInstr{LoadInstr{reg: r4, value: &RegisterLoadOperand{reg: r0}}}

	insch <- &ADDInstr{BaseBinaryInstr{dest: r5, lhs: r0,
		rhs: ImmediateOperand{4}}}

	insch <- &LABELInstr{mPrintStringLoopLabel}

	insch <- &TEQInstr{BaseComparisonInstr{lhs: r4, rhs: &ImmediateOperand{0}}}

	insch <- &BInstr{cond: condEQ, label: mPrintStringEndLabel}

	insch <- &LDRInstr{LoadInstr{reg: r0, value: &RegisterLoadOperand{reg: r5}}}

	insch <- &BLInstr{BInstr{label: mPutChar}}

	insch <- &SUBInstr{BaseBinaryInstr{dest: r4, lhs: r4,
		rhs: ImmediateOperand{1}}}

	insch <- &ADDInstr{BaseBinaryInstr{dest: r5, lhs: r5,
		rhs: ImmediateOperand{4}}}

	insch <- &BInstr{label: mPrintStringLoopLabel}

	insch <- &LABELInstr{mPrintStringEndLabel}

	insch <- &MOVInstr{dest: r0, source: &ImmediateOperand{n: 0}}

	insch <- &BLInstr{BInstr{label: mFFlush}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{r4, r5}}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//printIntgenerates code to print a given int
// p_print_int:
// -->	PUSH {lr}
// -->	MOV r1, r0
// -->	LDR r0, =msg_0
// -->	ADDS r0, r0, #4
// -->	BL printf
// -->	MOV r0, #0
// -->	BL fflush
// -->	POP {pc}
func printInt(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mPrintInt)

	insch <- &LABELInstr{mPrintIntLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &MOVInstr{dest: r1, source: r0}

	insch <- &LDRInstr{LoadInstr{reg: r0,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: r0, lhs: r0,
		rhs: ImmediateOperand{n: 4}}}

	insch <- &BLInstr{BInstr{label: mPrintf}}

	insch <- &MOVInstr{dest: r0, source: &ImmediateOperand{n: 0}}

	insch <- &BLInstr{BInstr{label: mFFlush}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//printChar code to print a given char
// p_print_char:
// -->	PUSH {lr}
// -->	BL putchar
// -->	POP {pc}
func printChar(alloc *RegAllocator, insch chan<- Instr) {
	insch <- &LABELInstr{mPrintCharLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &BLInstr{BInstr{label: mPutChar}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//printBool code to print a given bool
// p_print_bool:
// -->	PUSH {lr}
// -->	CMP r0, #0
// -->	LDRNE r0, =msg_1
// -->	LDREQ r0, =msg_2
// -->	ADDS r0, r0, #4
// -->	BL printf
// -->	MOV r0, #0
// -->	BL fflush
// -->	POP {pc}
func printBool(alloc *RegAllocator, insch chan<- Instr) {
	msg0 := alloc.stringPool.Lookup8(mTrue)
	msg1 := alloc.stringPool.Lookup8(mFalse)

	insch <- &LABELInstr{mPrintBoolLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &CMPInstr{BaseComparisonInstr{lhs: r0,
		rhs: &ImmediateOperand{n: 0}}}

	insch <- &LDRInstr{LoadInstr{reg: r0, cond: condNE,
		value: &BasicLoadOperand{value: msg0}}}

	insch <- &LDRInstr{LoadInstr{reg: r0, cond: condEQ,
		value: &BasicLoadOperand{value: msg1}}}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: r0, lhs: r0,
		rhs: ImmediateOperand{n: 4}}}

	insch <- &BLInstr{BInstr{label: mPrintf}}

	insch <- &MOVInstr{dest: r0, source: &ImmediateOperand{n: 0}}

	insch <- &BLInstr{BInstr{label: mFFlush}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//printReference code to print a given reference
// p_print_reference:
// -->	PUSH {lr}
// -->	MOV r1, r0
// -->	LDR r0, =msg_3
// -->	ADDS r0, r0, #4
// -->	BL printf
// -->	MOV r0, #0
// -->	BL fflush
// -->	POP {pc}
func printReference(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mPrintReference)

	insch <- &LABELInstr{mPrintReferenceLabel}

	insch <- &PUSHInstr{BaseStackInstr{regs: []Reg{lr}}}

	insch <- &MOVInstr{dest: r1, source: r0}

	insch <- &LDRInstr{LoadInstr{reg: r0,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &ADDInstr{BaseBinaryInstr{dest: r0, lhs: r0,
		rhs: &ImmediateOperand{4}}}

	insch <- &BLInstr{BInstr{label: mPrintf}}

	insch <- &MOVInstr{dest: r0, source: ImmediateOperand{0}}

	insch <- &BLInstr{BInstr{label: mFFlush}}

	insch <- &POPInstr{BaseStackInstr{regs: []Reg{pc}}}
}

//readInt code to read a given int
// p_read_int:
// -->	PUSH {lr}
// -->	MOV r1, r0
// -->	LDR r0, =msg_5
// -->	ADDS r0, r0, #4
// -->	BL scanf
// -->	POP {pc}
func readInt(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mPrintInt)

	insch <- &LABELInstr{mReadIntLabel}

	insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

	insch <- &MOVInstr{dest: r1, source: r0}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r0,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &ADDInstr{
		BaseBinaryInstr: BaseBinaryInstr{dest: r0, lhs: r0,
			rhs: &ImmediateOperand{n: 4}}}

	insch <- &BLInstr{BInstr: BInstr{label: mScanf}}

	insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}
}

//readChar code to read a given char
// p_read_char:
// -->	PUSH {lr}
// -->	MOV r1, r0
// -->	LDR r0, =msg_6
// -->	ADDS r0, r0, #4
// -->	BL scanf
// -->	POP {pc}
func readChar(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mReadChar)

	insch <- &LABELInstr{mReadCharLabel}

	insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

	insch <- &MOVInstr{dest: r1, source: r0}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r0,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: r0, lhs: r0,
		rhs: &ImmediateOperand{n: 4}}}

	insch <- &BLInstr{BInstr: BInstr{label: mScanf}}

	insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}
}

//checkDivideByZero code to check if a divide by zero occurs
// p_check_divide_by_zero:
// -->	PUSH {lr}
// -->	CMP r1, #0
// -->	LDREQ r0, =msg_7
// -->	BLEQ p_throw_runtime_error
// -->	POP {pc}
func checkDivideByZero(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mDivideByZeroErr)
	alloc.fsPool.Add(mThrowRuntimeErr)

	insch <- &LABELInstr{ident: mDivideByZeroLbl}

	insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

	insch <- &CMPInstr{BaseComparisonInstr: BaseComparisonInstr{lhs: r1,
		rhs: &ImmediateOperand{n: 0}}}

	insch <- &LDRInstr{LoadInstr{reg: r0, cond: condEQ,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &BLInstr{BInstr: BInstr{cond: condEQ, label: mThrowRuntimeErr}}

	insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}

}

//checkNullPointer code to check if it is a null pointer
// pi_check_null_pointer:
// -->	PUSH {lr}
// -->	CMP r0, #0
// -->	LDREQ r0, =msg_8
// -->	BLEQ p_throw_runtime_error
// -->	POP {pc}
func checkNullPointer(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mNullReferenceErr)
	alloc.fsPool.Add(mThrowRuntimeErr)

	insch <- &LABELInstr{ident: mNullReferenceLbl}

	insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

	insch <- &CMPInstr{BaseComparisonInstr: BaseComparisonInstr{lhs: r0,
		rhs: &ImmediateOperand{n: 0}}}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r0, cond: condEQ,
		value: &BasicLoadOperand{value: msg}}}

	insch <- &BLInstr{BInstr: BInstr{cond: condEQ, label: mThrowRuntimeErr}}

	insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}
}

//checkArrayBounds code to check if an Array Elem is in bounds
// p_check_array_bounds:
// -->	PUSH {lr}
// -->	CMP r0, #0
// -->	LDRLT r0, =msg_9
// -->	BLLT p_throw_runtime_error
// -->	LDR r1, [r1]
// -->	CMP r0, r1
// -->	LDRCS r0, =msg_10
// -->	BLCS p_throw_runtime_error
// -->	POP {pc}
func checkArrayBounds(alloc *RegAllocator, insch chan<- Instr) {
	msg0 := alloc.stringPool.Lookup8(mArrayNegIndexErr)
	msg1 := alloc.stringPool.Lookup8(mArrayLrgIndexErr)
	alloc.fsPool.Add(mThrowRuntimeErr)

	insch <- &LABELInstr{ident: mArrayBoundLbl}

	insch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

	insch <- &CMPInstr{BaseComparisonInstr: BaseComparisonInstr{lhs: r0,
		rhs: &ImmediateOperand{n: 0}}}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r0, cond: condLT,
		value: &BasicLoadOperand{value: msg0}}}

	insch <- &BLInstr{BInstr: BInstr{cond: condLT, label: mThrowRuntimeErr}}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r1,
		value: &RegisterLoadOperand{reg: r1}}}

	insch <- &CMPInstr{BaseComparisonInstr: BaseComparisonInstr{lhs: r0, rhs: r1}}

	insch <- &LDRInstr{LoadInstr: LoadInstr{reg: r0, cond: condCS,
		value: &BasicLoadOperand{value: msg1}}}

	insch <- &BLInstr{BInstr: BInstr{cond: condCS, label: mThrowRuntimeErr}}

	insch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}
}

//checkOverflowUnderflow code to check if operation is in under/overflow
// p_throw_overflow_error:
// -->	LDR r0, =msg_11
// -->	BL p_throw_runtime_error
func checkOverflowUnderflow(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mOverflowErr)
	alloc.fsPool.Add(mThrowRuntimeErr)

	insch <- &LABELInstr{ident: mOverflowLbl}

	insch <- &LDRInstr{
		LoadInstr: LoadInstr{reg: r0, value: &BasicLoadOperand{value: msg}},
	}

	insch <- &BLInstr{BInstr: BInstr{label: mThrowRuntimeErr}}
}

//throwRuntimeError throws a runtime error
// p_throw_runtime_error:
// -->	LDR r1, [r0]
// -->	ADDS r2, r0, #4
// -->	LDR r0, =msg_12
// -->	ADDS r0, r0, #4
// -->	BL printf
// -->	MOV r0, #0
// -->	BL fflush
// -->	MOV r0, #-1
// -->	BL exit
func throwRuntimeError(alloc *RegAllocator, insch chan<- Instr) {
	msg := alloc.stringPool.Lookup8(mPrintString)

	insch <- &LABELInstr{ident: mThrowRuntimeErr}

	insch <- &LDRInstr{
		LoadInstr: LoadInstr{reg: r1, value: &RegisterLoadOperand{reg: r0}}}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: r2, lhs: r0,
		rhs: ImmediateOperand{n: 4}}}

	insch <- &LDRInstr{
		LoadInstr: LoadInstr{reg: r0, value: &BasicLoadOperand{value: msg}}}

	insch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: r0, lhs: r0,
		rhs: ImmediateOperand{n: 4}}}

	insch <- &BLInstr{BInstr: BInstr{label: mPrintf}}

	insch <- &MOVInstr{dest: r0, source: &ImmediateOperand{n: 0}}

	insch <- &BLInstr{BInstr: BInstr{label: mFFlush}}

	insch <- &MOVInstr{dest: r0, source: &ImmediateOperand{n: -1}}

	insch <- &BLInstr{BInstr: BInstr{label: mExitLabel}}
}

//------------------------------------------------------------------------------
// GENERAL CODEGEN UTILITY
//------------------------------------------------------------------------------

func codeGenBuiltin(strPool *StringPool, fsPool *FSPool, f func(*RegAllocator, chan<- Instr)) <-chan Instr {
	ch := make(chan Instr)

	alloc := CreateRegAllocator()
	alloc.stringPool = strPool
	alloc.fsPool = fsPool

	go func() {
		f(alloc, ch)
		close(ch)
	}()

	return ch
}

// CodeGen generates instructions for functions
func (m *FunctionDef) CodeGen(strPool *StringPool, fsPool *FSPool) <-chan Instr {
	ch := make(chan Instr)

	go func() {
		alloc := CreateRegAllocator()
		alloc.stringPool = strPool
		alloc.fsPool = fsPool
		alloc.fname = m.ident

		ch <- &LABELInstr{m.ident}

		alloc.StartScope(ch)

		// save previous pc for returning
		ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{lr}}}

		ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{ip}}}

		// save callee saved registers
		ch <- &PUSHInstr{
			BaseStackInstr: BaseStackInstr{
				regs: []Reg{r4, r5, r6, r7, r8, r9, r10, r11},
			},
		}

		// put the first four params on the stack
		pl := len(m.params)
		switch {
		case pl >= 4:
			ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{r3}}}
			fallthrough
		case pl == 3:
			ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{r2}}}
			fallthrough
		case pl == 2:
			ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{r1}}}
			fallthrough
		case pl == 1:
			ch <- &PUSHInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{r0}}}
		}

		// set the addresses of the arguments relative to sp on the
		// stack
		for i := 0; i < 4 && i < len(m.params); i++ {
			p := m.params[i]
			alloc.stack[0][p.name] = i * -4
		}

		for i := 4; i < len(m.params); i++ {
			p := m.params[i]
			alloc.stack[0][p.name] = -4 + -4 + i*-4 + 8*-4
		}

		alloc.StartScope(ch)

		// codegen the function body
		m.body.CodeGen(alloc, ch)

		alloc.CleanupScope(ch)

		// if the function has no return type then zero r0 before
		// returning
		switch m.returnType.(type) {
		case InvalidType:
			ch <- &MOVInstr{dest: resReg, source: ImmediateOperand{0}}
		}

		ch <- &LABELInstr{fmt.Sprintf("%s_return", m.ident)}

		// restore the stack from pushing first four parameters
		if pl := len(m.params); pl > 0 {
			ppregs := pl * 4
			if ppregs > 16 {
				ppregs = 16
			}
			ch <- &ADDInstr{BaseBinaryInstr: BaseBinaryInstr{dest: sp, lhs: sp,
				rhs: ImmediateOperand{ppregs}}}
		}

		// restore callee saved registers
		ch <- &POPInstr{
			BaseStackInstr: BaseStackInstr{
				regs: []Reg{r4, r5, r6, r7, r8, r9, r10, r11},
			},
		}

		ch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{ip}}}

		// return
		ch <- &POPInstr{BaseStackInstr: BaseStackInstr{regs: []Reg{pc}}}

		// ensures literal pools for LDR are in range
		ch <- &LTORGInstr{}

		close(ch)
	}()

	return ch
}

var FSMap = map[string]func(*RegAllocator, chan<- Instr){
	mPrintIntLabel:       printInt,
	mPrintCharLabel:      printChar,
	mPrintBoolLabel:      printBool,
	mPrintStringLabel:    printString,
	mPrintReferenceLabel: printReference,
	mPrintNewLineLabel:   printNewLine,
	mReadIntLabel:        readInt,
	mReadCharLabel:       readChar,
	mDivideByZeroLbl:     checkDivideByZero,
	mNullReferenceLbl:    checkNullPointer,
	mArrayBoundLbl:       checkArrayBounds,
	mOverflowLbl:         checkOverflowUnderflow,
	mThrowRuntimeErr:     throwRuntimeError,
}

// CodeGen generates instructions for the whole program
func (m *AST) CodeGen() <-chan Instr {
	ch := make(chan Instr)
	var charr []<-chan Instr

	strPool := &StringPool{}
	fsPool := &FSPool{}

	// start codegen for all functions concurrently
	for _, f := range m.functions {
		charr = append(charr, f.CodeGen(strPool, fsPool))
	}
	mainF := &FunctionDef{
		ident:      "main",
		returnType: InvalidType{},
		body:       m.main,
	}
	charr = append(charr, mainF.CodeGen(strPool, fsPool))

	go func() {
		ch <- &DataSegInstr{}

		// buffer all the text instructions so the global stringpool
		// is filled
		var txtInstr []Instr
		txtInstr = append(txtInstr, &TextSegInstr{})
		txtInstr = append(txtInstr, &GlobalInstr{"main"})

		for _, fch := range charr {
			for instr := range fch {
				txtInstr = append(txtInstr, instr)
			}
		}

		// generate code for builtin functions
		// prints, reads, runtime errors
		for function, print := range fsPool.pool {
			if print {
				for instr := range codeGenBuiltin(strPool, fsPool, FSMap[function]) {
					txtInstr = append(txtInstr, instr)
				}
			}
		}

		// output the strings used in the WACC program
		for i := 0; i < len(strPool.pool); i++ {
			v := strPool.pool[i]
			ch <- &LABELInstr{fmt.Sprintf("msg_%d", i)}
			ch <- &DataWordInstr{v.len}
			ch <- &DataASCIIInstr{v.str}
		}

		// output the instructions
		for _, tin := range txtInstr {
			ch <- tin
		}

		close(ch)
	}()

	return ch
}
