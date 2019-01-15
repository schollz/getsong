package getsong

import (
	"fmt"
	"strings"
	"testing"

	log "github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

func TestGetSongAPI(t *testing.T) {
	defer log.Flush()

	_, err := GetSong("Old Records", "Allen Toussaint", Options{
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
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

	// this one is tricky because the band name is spelled weird and requires
	// clicking through to force youtube to search the wrong spelling
	id, err := getMusicVideoID("eva", "haerts")
	assert.Nil(t, err)
	assert.Equal(t, "qxiOMm_x3Xg", id)

	// this one is trick because its the second result
	id, err = getMusicVideoID("old records", "allen toussaint")
	assert.Nil(t, err)
	assert.Equal(t, "o3HtZ4qIkaI", id)
}

func TestGetRenderedPage(t *testing.T) {
	defer log.Flush()
	setLogLevel("debug")
	html, err := getRenderedPage("https://www.youtube.com/")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(html, "recommended"))
}
