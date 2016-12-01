package main

import (
	"github.com/owulveryck/echoprint-codegen/whitening"
	"os"
)

func main() {
	w := whitening.NewWriter(os.Stdout)
	w.ReadFrom(os.Stdin)
}
