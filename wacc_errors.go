package main

import (
	"fmt"
	"os"
)

type WaccError struct {
	exitCode int
	err      error
}

func errorHandler(wErr WaccError) {
	fmt.Println(wErr.err)
	os.Exit(wErr.exitCode)
}
