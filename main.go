package main

import (
	"encoding/binary"
	"github.com/owulveryck/echoprint-codegen/whitening"
	"log"
	"os"
)

func main() {
	// Read n samples from stdin
	n := 100000
	data := make([]float32, n)
	// Read the flow from stdin
	if err := binary.Read(os.Stdin, binary.LittleEndian, data); err != nil {
		log.Fatal(err)
	}
	w := whitening.NewWhitening(&data, uint64(n))
	w.Compute()
}
