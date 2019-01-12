package getsong

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSong(t *testing.T) {
	SetLogLevel("debug")

	id, err := GetMusicVideoIDs("transmission listen helado negro", 224)
	assert.Nil(t, err)
	assert.Equal(t, "d", id)
	fname, err := DownloadYouTube("JkIM2xp65B8", "Helado Negro - Transmission")
	assert.Nil(t, err)
	assert.Nil(t, ConvertToMp3(fname))
}
