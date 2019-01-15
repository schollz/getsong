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
	fname, err := getsong.GetSong(getsong.Options{
		Title:        os.Args[1],
		Artist:       os.Args[2],
		ShowProgress: true,
		Debug:        true,
	})
	if err == nil {
		fmt.Printf("Downloaded '%s'\n", fname)
	}
}
