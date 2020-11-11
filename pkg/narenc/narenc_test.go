package narenc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func skippable(entry os.FileInfo) bool {

	return false ||
		entry.Name() == ".links" ||
		strings.HasSuffix(entry.Name(), ".drv") ||
		strings.HasSuffix(entry.Name(), ".lock") ||
		strings.HasSuffix(entry.Name(), ".zip") ||
		strings.HasSuffix(entry.Name(), ".tar.gz") ||
		strings.HasSuffix(entry.Name(), ".tar.xz") ||
		strings.HasSuffix(entry.Name(), "-source")
}

func TestRandomNStorePaths(t *testing.T) {
	d := "/nix/store"
	entries, err := ioutil.ReadDir(d)
	if err != nil {
		t.Fatal(err)
	}

	count := 10

	for i := 0; i < count; i++ {
		entry := entries[rand.Intn(len(entries))]
		if skippable(entry) {
			count++
			continue
		}
		log.Println("checking entry", entry)
		pth := filepath.Join(d, entry.Name())

		c := make(chan []byte, 1)
		go func() {
			dumpCmd := exec.Command("nix", "dump-path", pth)
			log.Println(dumpCmd)
			if err != nil {
				panic(err)
			}
			dumpedNarBytes, err := dumpCmd.Output()
			if err != nil {
				panic(err)
			}
			c <- dumpedNarBytes
		}()

		var myNarBytesBuf bytes.Buffer
		err = Encode(&myNarBytesBuf, pth)
		if err != nil {
			t.Fatal(err)
		}
		myNarBytes := myNarBytesBuf.Bytes()

		dumpedNarBytes := <-c

		if !reflect.DeepEqual(dumpedNarBytes, myNarBytes) {
			t.Fatal("didn't match", entry.Name())
		}
	}
}

// TODO: this test relies on paths in the store
func TestNarEncoding(t *testing.T) {
	expectedFile, err := os.Open("./testdata/a5546jp1hl164cklp4271rjavgacn0p7.nar.xz")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reader, err := xz.NewReader(expectedFile)
	if err != nil {
		t.Fatal(err)
	}

	n, err := io.Copy(&buf, reader)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(n)
	expectedBytes := buf.Bytes()

	actualBuf := bytes.Buffer{}
	err = Encode(&actualBuf, "/nix/store/a5546jp1hl164cklp4271rjavgacn0p7-hello-2.10")
	if err != nil {
		t.Fatal(err)
	}
	actualBytes := actualBuf.Bytes()

	actualLen := len(actualBytes)
	expectedLen := len(expectedBytes)
	if expectedLen != actualLen {
		t.Fatalf("length wrong; actual=%d expected=%d", actualLen, expectedLen)
	}

	if !reflect.DeepEqual(expectedBytes, actualBytes) {
		t.Fatal("nar was wrong")
	}
}
