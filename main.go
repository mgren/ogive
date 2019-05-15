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