# getsong

[![travis](https://travis-ci.org/schollz/getsong.svg?branch=master)](https://travis-ci.org/schollz/getsong) 
[![go report card](https://goreportcard.com/badge/github.com/schollz/getsong)](https://goreportcard.com/report/github.com/schollz/getsong) 
[![coverage](https://img.shields.io/badge/coverage-46%25-yellow.svg)](https://gocover.io/github.com/schollz/getsong)
[![godocs](https://godoc.org/github.com/schollz/getsong?status.svg)](https://godoc.org/github.com/schollz/getsong) 

This is a simple library that utilizes the [rylio/ytdl YouTube downloaded](https://github.com/rylio/ytdl) and [ffmpeg](https://www.ffmpeg.org/) to get almost any mp3 of a song that you want to listen to. I wrote this so I could [download mp3s of my favorite Spotify playlists](https://github.com/schollz/spotifydownload).

## Install

```
go get -u github.com/schollz/getsong/...
```

## Usage 

### Use as a program

```bash
$ getsong 'Getting in Tune by The Who'

```

### Use as a library

```golang
fname, err := getsong.GetSong(getsong.Options{
    Title:        "True",
    Artist:       "Spandau Ballet",
})
if err == nil {
    fmt.Printf("Downloaded '%s'\n", fname)
}
```

## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## License

MIT
