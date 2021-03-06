package main

// WACC Group 34
//
// flags.go: Parses the different flags added when running ./wacc_34
//
// File contains functions that parse the flags, and if detected, handle their
// expected behaviour

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

// Flags structure contains all the flag values and the filename
type Flags struct {
	filename      string
	assemblyfile  string
	printPEGTree  bool
	printPretty   bool
	printAST      bool
	verbose       bool
	printAssembly bool
	noassembly    bool
	optimise      bool
}

// Parse defines all the flags and then parses the command line args
func (f *Flags) Parse() {
	flag.StringVar(&f.filename, "file", "", "Input File")

	flag.BoolVar(&f.printPEGTree, "peg", false,
		"Print PEG tree for the supplied file")
	flag.BoolVar(&f.printPretty, "pretty", false,
		"Pretty print the supplied file")
	flag.BoolVar(&f.printAST, "ast", false,
		"Print AST for the supplied file")
	flag.BoolVar(&f.verbose, "verbose", false,
		"Print different stages during compilation")
	flag.BoolVar(&f.printAssembly, "assembly", false,
		"Print assembly instructions to STD Output")
	flag.BoolVar(&f.noassembly, "noassembly", false,
		"Assembly file not produced, no assembly to STD Output")
	flag.BoolVar(&f.optimise, "optimise", false,
		"Optimise the AST generated from the WACC file")

	flag.Parse()

	f.assemblyfile = filepath.Base(
		strings.TrimSuffix(
			f.filename,
			filepath.Ext(f.filename),
		) + ".s",
	)
}

// Start prints compiling message when verbose flag is set
func (f *Flags) Start() {
	if f.verbose {
		fmt.Println("-- Compiling...")
	}
}

// Finish prints finished message when verbose flag is set
func (f *Flags) Finish() {
	if f.verbose {
		fmt.Println("-- Finished")
	}
}

// PrintPrettyAST will pretty print the code or print the AST when appropriate
// flags are supplied
func (f *Flags) PrintPrettyAST(ast *AST) {
	if f.printPretty {
		fmt.Println("-- Printing Pretty Code")
		fmt.Println(ast)
	}

	if f.printAST {
		fmt.Println("-- Printing AST")
		fmt.Println(ast.aststring())
	}
}
