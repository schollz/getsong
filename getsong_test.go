package getsong

import (
	"fmt"
	"os"
	"testing"

	log "github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

func TestGetSongAPI(t *testing.T) {
	defer log.Flush()

	fname, err := GetSong(Options{
		Title:        "True",
		Artist:       "Spandau Ballet",
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
	assert.Equal(t, "Spandau Ballet - True.mp3", fname)
}
func TestGetSong(t *testing.T) {
	defer log.Flush()
	optionShowProgressBar = true

	id, err := getMusicVideoID("transmission listen helado negro", 224)
	assert.Nil(t, err)
	assert.Equal(t, "JkIM2xp65B8", id)
	fname, err := downloadYouTube(id, "Helado Negro - Transmission")
	assert.Nil(t, err)
	assert.Nil(t, convertToMp3(fname))
	os.Remove(fname)
}

func TestGetFfmpeg(t *testing.T) {
	defer log.Flush()
	optionShowProgressBar = true

	locationToBinary, err := getFfmpegBinary()
	fmt.Println(locationToBinary)
	assert.NotEqual(t, "", locationToBinary)
	assert.Nil(t, err)
}
