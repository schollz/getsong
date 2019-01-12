package getsong

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	log "github.com/cihub/seelog"
	"github.com/otium/ytdl"
)

func init() {
	SetLogLevel("info")

}

// SetLogLevel determines the log level
func SetLogLevel(level string) (err error) {
	// https://github.com/cihub/seelog/wiki/Log-levels
	appConfig := `
	<seelog minlevel="` + level + `">
	<outputs formatid="stdout">
	<filter levels="debug,trace">
		<console formatid="debug"/>
	</filter>
	<filter levels="info">
		<console formatid="info"/>
	</filter>
	<filter levels="critical,error">
		<console formatid="error"/>
	</filter>
	<filter levels="warn">
		<console formatid="warn"/>
	</filter>
	</outputs>
	<formats>
		<format id="stdout"   format="%Date %Time [%LEVEL] %File %FuncShort:%Line %Msg %n" />
		<format id="debug"   format="%Date %Time %EscM(37)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="info"    format="%Date %Time %EscM(36)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="warn"    format="%Date %Time %EscM(33)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="error"   format="%Date %Time %EscM(31)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
	</formats>
	</seelog>
	`
	logger, err := log.LoggerFromConfigAsBytes([]byte(appConfig))
	if err != nil {
		return
	}
	log.ReplaceLogger(logger)
	return
}

// ConvertToMp3 uses ffmpeg to convert to mp3
func ConvertToMp3(filename string) (err error) {
	filenameWithoutExtension := strings.TrimRight(filename, filepath.Ext(filename))
	// convert to mp3
	cmd := exec.Command("ffmpeg", "-i", filename, "-y", filenameWithoutExtension+".mp3")
	_, err = cmd.CombinedOutput()
	return
}

// DownloadYouTube downloads a youtube video and saves using the filename. Returns the filename with the extension.
func DownloadYouTube(youtubeID string, filename string) (downloadedFilename string, err error) {
	info, err := ytdl.GetVideoInfo(youtubeID)
	if err != nil {
		err = fmt.Errorf("Unable to fetch video info: %s", err.Error())
		return
	}
	bestQuality := 0
	var format ytdl.Format
	for _, f := range info.Formats {
		if f.VideoEncoding == "" {
			if f.AudioBitrate > bestQuality {
				bestQuality = f.AudioBitrate
				format = f
			}
		}
	}
	if bestQuality == 0 {
		err = fmt.Errorf("No audio available")
		return
	}
	downloadURL, err := info.GetDownloadURL(format)
	log.Debugf("downloading %s", downloadURL)
	if err != nil {
		err = fmt.Errorf("Unable to get download url: %s", err.Error())
		return
	}

	var out io.Writer
	saveFile, err := os.Create(fmt.Sprintf("%s.%s", filename, format.Extension))
	if err != nil {
		return
	}
	downloadedFilename = saveFile.Name()
	out = saveFile
	log.Debugf("downloading %s to %s", info.Title, saveFile.Name())

	var req *http.Request
	req, err = http.NewRequest("GET", downloadURL.String(), nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if err == nil {
			err = fmt.Errorf("Received status code %d from download url", resp.StatusCode)
		}
		err = fmt.Errorf("Unable to start download: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	progressBar := pb.New64(resp.ContentLength)
	progressBar.SetUnits(pb.U_BYTES)
	progressBar.ShowTimeLeft = true
	progressBar.ShowSpeed = true
	progressBar.RefreshRate = 1 * time.Second
	progressBar.Output = os.Stderr
	progressBar.Start()
	defer progressBar.Finish()
	out = io.MultiWriter(out, progressBar)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return
	}

	return
}

// GetMusicVideoIDs returns the ids for a specified title and artist
func GetMusicVideoIDs(titleAndArtist string, expectedDuration ...int) (id string, err error) {
	youtubeSearchURL := fmt.Sprintf(
		`https://www.youtube.com/results?search_query="Provided+to+YouTube"+%s`,
		strings.Join(strings.Fields(titleAndArtist), "+"),
	)
	log.Debugf("searching url: %s", youtubeSearchURL)

	client := &http.Client{}

	req, err := http.NewRequest("GET", youtubeSearchURL, nil)
	if err != nil {
		log.Error(err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return
	}

	// do this now so it won't be forgotten
	defer resp.Body.Close()
	// reads html as a slice of bytes
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "Provided to YouTube") {
			continue
		}
		if !strings.Contains(line, "yt-lockup-title") {
			continue
		}
		durationParts := strings.Split(getStringInBetween(line, "Duration: ", "."), ":")
		if len(durationParts) != 2 {
			continue
		}
		minutes, errExtract := strconv.Atoi(durationParts[0])
		if errExtract != nil {
			log.Error(errExtract)
			continue
		}
		seconds, errExtract := strconv.Atoi(durationParts[1])
		if errExtract != nil {
			log.Error(errExtract)
			continue
		}
		youtubeID := getStringInBetween(line, `/watch?v=`, `"`)
		youtubeTitle := getStringInBetween(line, `title="`, `"`)
		youtubeDuration := minutes*60 + seconds
		if len(expectedDuration) > 0 {
			if math.Abs(float64(expectedDuration[0]-youtubeDuration)) > 10 {
				log.Debugf("'%s' duration (%ds) is different than expected (%ds)", youtubeTitle, youtubeDuration, expectedDuration[0])
				continue
			}
		}
		log.Debugf("%s (%s): %ds", youtubeTitle, youtubeID, youtubeDuration)
		id = youtubeID
		return
	}
	err = fmt.Errorf("could not find any videos that matched")
	return
}

// getStringInBetween Returns empty string if no start string found
func getStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str[s:], end)
	return str[s : s+e]
}

var illegalFileNameCharacters = regexp.MustCompile(`[^[a-zA-Z0-9]-_]`)

func sanitizeFileNamePart(part string) string {
	part = strings.Replace(part, "/", "-", -1)
	part = illegalFileNameCharacters.ReplaceAllString(part, "")
	return part
}
