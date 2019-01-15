package main

import (
	"fmt"
	"os"

	"github.com/schollz/getsong"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println(`Usage:
		
getsong 'title of song' 'artist of song'

`)
		os.Exit(1)
	}
	fname, err := getsong.GetSong(os.Args[1], os.Args[2], getsong.Options{
		ShowProgress: true,
	})
	if err == nil {
		fmt.Printf("Downloaded '%s'\n", fname)
	} else {
		fmt.Println(err)
	}
}
