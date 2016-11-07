package main

import (
	"errors"
	"flag"
)

type Flags struct {
	filename  string
	printTree bool
}

func (f *Flags) Parse() {
	flag.BoolVar(&f.printTree, "t", false, "Print AST for the supplied file")
	flag.Parse()

	if len(flag.Args()) == 0 {
		errorHandler(WaccError{err: errors.New("Error: Input file missing"), exitCode: invalidArgError})
	}

	f.filename = flag.Args()[0]
}
