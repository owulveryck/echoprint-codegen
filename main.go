package main

import (
	"github.com/owulveryck/echoprint-codegen/dsp"
	"github.com/owulveryck/echoprint-codegen/whitening"
	"os"
)

func main() {
	s := whitening.NewWhitener()
	w := dsp.NewWriter(os.Stdout, s)
	w.ReadFrom(os.Stdin)
}
