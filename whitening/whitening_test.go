package whitening_test

import (
	"encoding/binary"
	"github.com/owulveryck/echoprint-codegen/whitening"
	"io"
	"log"
	"os"
	"os/exec"
)

func ExampleReader_Read() {
	file, err := os.Open("../samples/Largo+from+-Concerto-No5_JS_Bach.pcm")
	if err != nil {
		log.Fatal("Cannot open test file: ", err)
	}
	white := whitening.NewReader(file, 1, 11025, 16, binary.LittleEndian)
	subProcess := exec.Command("play", "-t", "raw", "-r", "11025", "-e", "signed", "-b", "16", "-c", "1", "-")
	stdin, err := subProcess.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdin.Close() // the doc says subProcess.Wait will close it, but I'm not sure, so I kept this line

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil { //Use start, not run
		log.Fatal(err)
	}

	io.Copy(stdin, white)
	subProcess.Wait()
	// Output:

}
