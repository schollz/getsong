package getsong

import (
	"fmt"
	"strings"
	"testing"

	log "github.com/schollz/logger"
	"github.com/stretchr/testify/assert"
)

func TestGetSongAPI(t *testing.T) {
	_, err := GetSong("Old Records", "Allen Toussaint", Options{
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
	_, err = GetSong("Eva", "Haerts", Options{
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
}

func TestGetPage(t *testing.T) {

	log.SetLevel("debug")
	html, err := getPage("https://www.youtube.com/watch?v=qxiOMm_x3Xg")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(html, "<html"))
}
func TestGetFfmpeg(t *testing.T) {

	OptionShowProgressBar = true

	locationToBinary, err := getFfmpegBinary()
	fmt.Println(locationToBinary)
	assert.NotEqual(t, "", locationToBinary)
	assert.Nil(t, err)
}

func TestGetYouTubeInfo(t *testing.T) {
	log.SetLevel("debug")
	info, err := getYoutubeVideoInfo("qxiOMm_x3Xg")
	assert.Nil(t, err)
	fmt.Printf("%+v\n", info)
}

func TestOne(t *testing.T) {
	_, err := GetSong("Old Records", "Allen Toussaint", Options{
		ShowProgress: true,
		Debug:        true,
	})
	assert.Nil(t, err)
}

func TestGetMusicVideoID(t *testing.T) {
	log.SetLevel("trace")

	// this one is tricky because the band name is spelled weird and requires
	// clicking through to force youtube to search the wrong spelling
	id, err := GetMusicVideoID("eva", "haerts")
	log.Infof("eva: %s", id)
	assert.Nil(t, err)
	assert.Equal(t, "qxiOMm_x3Xg", id)

	id, err = GetMusicVideoID("movies", "Weyes Blood")
	log.Infof("movies: %s", id)
	assert.Nil(t, err)
	assert.True(t, "RFtRq6t3jOo" == id || "xniRJsus8pk" == id)

	// this one is trick because its the second result
	id, err = GetMusicVideoID("old records", "allen toussaint")
	log.Infof("old records: %s", id)
	assert.Nil(t, err)
	assert.True(t, "oa6KzRfvtAs" == id || "obtJEJ4VPmk" == id)

	// try one with puncuation
	id, err = GetMusicVideoID("hey, ma", "bon iver")
	log.Infof("hey, ma: %s", id)
	assert.Nil(t, err)
	assert.True(t, "HDAKS18Gv1U" == id)

	// skip the most popular result to get the provided to youtube version
	id, err = GetMusicVideoID("true", "spandau ballet")
	log.Infof("true: %s", id)
	assert.Nil(t, err)
	assert.True(t, "ITX-SEsyGRg" == id || "2H1N6KdU-L0" == id || "sWBueqYA2Es" == id || "TVeSwMMvkP4" == id)

	// pick one that is not the first
	id, err = GetMusicVideoID("i know what love is", "don white")
	log.Infof("i know what: %s", id)
	assert.Nil(t, err)
	assert.Equal(t, "3LRu9mjiyKo", id)
}

func TestParseDurationString(t *testing.T) {
	assert.Equal(t, int64(470001), ParseDurationString("00:07:50.01,"))
}
