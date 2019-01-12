package getsong

import (
	"fmt"
	"os"
	"testing"

	log "github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

func TestGetSong(t *testing.T) {
	defer log.Flush()
	Debug(true)
	OptionShowProgressBar = true

	id, err := GetMusicVideoID("transmission listen helado negro", 224)
	assert.Nil(t, err)
	assert.Equal(t, "JkIM2xp65B8", id)
	fname, err := DownloadYouTube(id, "Helado Negro - Transmission")
	assert.Nil(t, err)
	assert.Nil(t, ConvertToMp3(fname))
	os.Remove(fname)
}

func TestGetFfmpeg(t *testing.T) {
	defer log.Flush()
	Debug(true)
	OptionShowProgressBar = true

	locationToBinary, err := getFfmpegBinary()
	fmt.Println(locationToBinary)
	assert.NotEqual(t, "", locationToBinary)
	assert.Nil(t, err)
}
