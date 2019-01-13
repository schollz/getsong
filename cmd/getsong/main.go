package main

import (
	"fmt"
	"os"

	"github.com/schollz/getsong"
)

func main() {
	fname, err := getsong.GetSong(getsong.Options{
		Title:        os.Args[1],
		ShowProgress: true,
	})
	if err == nil {
		fmt.Printf("Downloaded '%s'\n", fname)
	}
}
