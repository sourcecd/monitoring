package main

import (
	"fmt"
	"os"
)

func mulfunc(i int) (int, error) {
	return i * 2, nil
}

func main() {
	os.Exit(1) // want "direct os.Exit found in main/main"
	fmt.Println("Hello")

}

func okok() {
	os.Exit(0)
}

func TestFunc() {
	var i int
	myfunc := func() error {
		return nil
	}
	myfunc()
	if true {
		i := 7
		i, _ = mulfunc(i)
	}
	i, _ = i+1, myfunc()
}
