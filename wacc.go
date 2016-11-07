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

func errorHandler(err error, code int) {
	fmt.Println(err)
	os.Exit(code)
}

func main() {
	var flags Flags

	flags.Parse()

	filename := flags.filename

	file, err := os.Open(filename)

	fmt.Printf("%v: %v\n", time.Now().Format("2006/01/02 15:04:05"), filename)

	buffer, err := ioutil.ReadAll(file)
	if err != nil {
		errorHandler(err, fileReadError)
	}

	wacc := &WACC{Buffer: string(buffer)}
	wacc.Init()

	fmt.Println("-- Compiling...")

	if err := wacc.Parse(); err != nil {
		errorHandler(err, syntacticError)
	}

	if flags.printTree {
		fmt.Println("-- Abstract Syntax Tree")
		wacc.PrintSyntaxTree()
	}

	fmt.Println("\n-- Finished")
}
