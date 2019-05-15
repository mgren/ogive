package main

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/mgren/ogive/cmd"
)

func main() {
	memguard.CatchInterrupt(func() {
		fmt.Println("Exiting...")
	})

	defer memguard.DestroyAll()
	cmd.Execute()
}

// https://goreportcard.com/about/ + release + license
// https://blitiri.com.ar/git/r/chasquid/b/master/t/f=Makefile.html
// test z block device
// rebuild profile ze zmianą hasła / kluczy
// wytestować jeszcze raz
// review opisów poleceń w kodzie
// grep za Storage (powinno być deep archive)
