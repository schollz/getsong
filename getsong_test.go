package getsong

import (
	"fmt"
	"os"
	"strings"
	"testing"

	log "github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

func TestGetSongAPI(t *testing.T) {
	defer log.Flush()

	_, err := GetSong(Options{
		Title:        "Old Records",
		Artist:       "Allen Toussaint",
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
}
func TestGetSong(t *testing.T) {
	defer log.Flush()
	optionShowProgressBar = true

	id, err := getMusicVideoID("transmission listen", "helado negro", 224)
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

func TestGetYouTubeInfo(t *testing.T) {
	defer log.Flush()
	setLogLevel("debug")
	info, err := getYoutubeVideoInfo("qxiOMm_x3Xg")
	assert.Nil(t, err)
	fmt.Printf("%+v\n", info)
}

func TestGetMusicVideoID(t *testing.T) {
	defer log.Flush()
	setLogLevel("debug")
	id, err := getMusicVideoID("eva", "haerts")
	assert.Nil(t, err)
	assert.Equal(t, "qxiOMm_x3Xg", id)
}

func TestGetRenderedPage(t *testing.T) {
	defer log.Flush()
	setLogLevel("debug")
	html, err := getRenderedPage("https://www.youtube.com/")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(html, "recommended"))
}
