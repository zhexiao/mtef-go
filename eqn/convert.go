package eqn

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

func Convert(filepath string) string {
	buffer, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Print(err)
	}

	reader := bytes.NewReader(buffer)
	mtef, err := Open(reader)
	if err != nil {
		fmt.Println(err)
	}

	latex, err := mtef.Translate()
	if err != nil {
		fmt.Println(err)
	}
	return latex
}
