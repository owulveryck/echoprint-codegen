package main

import (
	"log"
	//"bytes"
	"encoding/binary"
	"io"
	"os"
)

func main() {
	p := make([]byte, 100000)
	l := len(p)

	//buf := bytes.NewBuffer(p)
	//var out []float32
	//out = make([]float32, 10000)
	var n int
	for n < l {
		var b []byte
		b = make([]byte, 2)
		var out int16
		//out = make(int16, 100)
		err := binary.Read(os.Stdin, binary.LittleEndian, &out)
		if err != nil {
			if err != io.EOF {
				log.Fatal("binary.Read failed:", err)
			}
			log.Println("All read")
			break
		}
		output := out / 2
		binary.LittleEndian.PutUint16(b, uint16(output))
		p[n] = b[0]
		p[n+1] = b[1]
		n += 2

	}
	err := binary.Write(os.Stdout, binary.LittleEndian, p)
	if err != nil {
		log.Fatal("binary.Write failed:", err)
	}
}
