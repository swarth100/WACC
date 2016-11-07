package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const invalidArgError int = 1
const fileReadError int = 2
const syntacticError int = 100
const symanticError int = 200
const timeFormat string = "2006/01/02 15:04:05"

func main() {
	var flags Flags
	flags.Parse()

	filename := flags.filename
	file, err := os.Open(filename)

	fmt.Printf("%v: %v\n", time.Now().Format("2006/01/02 15:04:05"), filename)

	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		errorHandler(WaccError{err: err, exitCode: fileReadError})
	}

	wacc := &WACC{Buffer: string(buffer)}
	wacc.Init()

	fmt.Println("-- Compiling...")

	if err := wacc.Parse(); err != nil {
		errorHandler(WaccError{err: err, exitCode: syntacticError})
	}

	if flags.printTree {
		printAST(wacc)
	}

	fmt.Println("-- Finished")
}

func printAST(wacc *WACC) {
	fmt.Println("-- Abstract Syntax Tree")
	wacc.PrintSyntaxTree()
}
