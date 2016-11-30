package whitening_test

import (
	"github.com/owulveryck/echoprint-codegen/whitening"
	"log"
	"os"
	"os/exec"
	"testing"
)

func TestReader(t *testing.T) {
	file, err := os.Open("../samples/Largo+from+-Concerto-No5_JS_Bach.pcm")
	if err != nil {
		t.Skipf("Cannot open sample file", err)
	}
	defer file.Close()
}

func BenchmarkReader(b *testing.B) {
	file, err := os.Open("../samples/Largo+from+-Concerto-No5_JS_Bach.pcm")
	if err != nil {
		b.Skipf("Cannot open sample file", err)
	}
	defer file.Close()
	for n := 0; n < b.N; n++ {

	}

}

func ExampleWriter() {
	file, err := os.Open("../samples/Largo+from+-Concerto-No5_JS_Bach.pcm")
	if err != nil {
		log.Fatal("Cannot open test file: ", err)
	}
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

	w := whitening.NewWriter(stdin)
	w.ReadFrom(file)
	subProcess.Wait()
	// Output:
}
