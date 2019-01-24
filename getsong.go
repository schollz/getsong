package getsong

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/otium/ytdl"
	"github.com/pkg/errors"
	pb "gopkg.in/cheggaaa/pb.v1"
)

const CHUNK_SIZE = 524288

var ffmpegBinary string
var optionShowProgressBar bool

func init() {
	setLogLevel("info")
	var err error
	ffmpegBinary, err = getFfmpegBinary()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Options allow you to set the artist, title and duration to find the right song.
// You can also set the progress and debugging for the program execution.
type Options struct {
	ShowProgress  bool
	Debug         bool
	DoNotDownload bool
}

// GetSong requires passing in the options which requires at least a title.
// If an Artist is provided, it will save it as Artist - Title.mp3
// You can also pass in a duration, and it will try to find a video that
// is within 10 seconds of that duration.
func GetSong(title string, artist string, option ...Options) (savedFilename string, err error) {
	defer log.Flush()
	var options Options
	if len(option) > 0 {
		options = option[0]
	}
	if options.Debug {
		setLogLevel("debug")
	} else {
		setLogLevel("info")
	}
	optionShowProgressBar = options.ShowProgress

	if title == "" {
		err = fmt.Errorf("must enter title")
		return
	}

	var youtubeID string
	youtubeID, err = getMusicVideoID(title, artist)
	if err != nil {
		err = errors.Wrap(err, "could not get youtube ID")
		return
	}
	youtubeID = strings.TrimSpace(youtubeID)
	if youtubeID == "" {
		err = fmt.Errorf("could not find youtube ID")
		return
	}

	if artist != "" {
		savedFilename = fmt.Sprintf("%s - %s (%s)", artist, title, youtubeID)
	} else {
		savedFilename = fmt.Sprintf("%s (%s)", title, youtubeID)
	}

	if !options.DoNotDownload {
		var fname string
		log.Debugf("trying to download 'https://www.youtube.com/watch?v=%s'", youtubeID)
		fname, err = downloadYouTube(youtubeID, savedFilename)
		if err != nil {
			err = errors.Wrap(err, "could not download video")
			return
		}

		if optionShowProgressBar {
			fmt.Println("...converting to mp3...")
		}
		err = convertToMp3(fname)
		if err != nil {
			err = errors.Wrap(err, "could not convert video")
			return
		}
	}

	savedFilename += ".mp3"
	return
}

// setLogLevel determines the log level
func setLogLevel(level string) (err error) {
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

// convertToMp3 uses ffmpeg to convert to mp3
func convertToMp3(filename string) (err error) {
	filenameWithoutExtension := strings.Replace(filename, filepath.Ext(filename), "", 1)
	// convert to mp3
	cmd := exec.Command(ffmpegBinary, "-i", filename, "-qscale:a", "3", "-y", filenameWithoutExtension+".mp3")
	_, err = cmd.CombinedOutput()
	if err == nil {
		os.Remove(filename)
	}
	return
}

// downloadYouTube downloads a youtube video and saves using the filename. Returns the filename with the extension.
func downloadYouTube(youtubeID string, filename string) (downloadedFilename string, err error) {
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
	downloadedFilename = fmt.Sprintf("%s.%s", filename, format.Extension)
	if optionShowProgressBar {
		fmt.Printf("Downloading %s...\n", filename)
	}

	err = downloadFromYouTube(downloadedFilename, downloadURL.String())
	return
}

func downloadFromYouTube(downloadedFilename string, downloadURL string) (err error) {
	// download in parallel
	// get the content length of the video
	respHead, err := http.Head(downloadURL)
	if err != nil {
		return
	}
	log.Debugf("total length: %d", respHead.ContentLength)
	contentLength := int(respHead.ContentLength)
	if contentLength > 15000000 {
		err = fmt.Errorf("content is to long: %d", contentLength)
		return
	}
	// split into ranges and download in parallel
	var wg sync.WaitGroup
	numberOfRanges := int(math.Ceil(float64(contentLength) / CHUNK_SIZE))
	for i := 0; i < numberOfRanges; i++ {
		startRange := i * CHUNK_SIZE
		endRange := startRange + CHUNK_SIZE
		if i != 0 {
			startRange += 1
		}
		if endRange > contentLength {
			endRange = contentLength
		}
		log.Debugf("%d-%d", startRange, endRange)
		wg.Add(1)
		go func(it, start, end int, wg *sync.WaitGroup, urlToGet string, downloadedFilename string) {
			defer wg.Done()
			var out io.Writer
			var f *os.File
			// open as write only
			f, err = os.OpenFile(fmt.Sprintf("%s%d", downloadedFilename, it), os.O_CREATE|os.O_WRONLY, 0666)
			if err != nil {
				log.Error(err)
				return
			}
			defer f.Close()
			out = f

			var req *http.Request
			req, err = http.NewRequest("GET", urlToGet, nil)
			partToGet := fmt.Sprintf("bytes=%d-%d", start, end)
			log.Debugf("%d getting part %s", it, partToGet)
			req.Header.Set("Range", partToGet)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Error(err)
				return
			}
			defer resp.Body.Close()
			if it == 0 && optionShowProgressBar {
				progressBar := pb.New64(resp.ContentLength)
				// progressBar.SetUnits(pb.U_BYTES)
				progressBar.ShowTimeLeft = true
				progressBar.ShowSpeed = true
				//	progressBar.RefreshRate = time.Millisecond * 1
				progressBar.Output = os.Stdout
				progressBar.Start()
				defer progressBar.Finish()
				out = io.MultiWriter(out, progressBar)
			}
			_, err = io.Copy(out, resp.Body)
		}(i, startRange, endRange, &wg, downloadURL, downloadedFilename)

	}
	wg.Wait()

	// concatanate
	f, err := os.OpenFile(downloadedFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Error(err)
		return
	}
	defer f.Close()
	for i := 0; i < numberOfRanges; i++ {
		fname := fmt.Sprintf("%s%d", downloadedFilename, i)
		fh, err := os.Open(fname)
		if err != nil {
			log.Error(err)
		}

		_, err = io.Copy(f, fh)
		if err != nil {
			log.Error(err)
		}
		fh.Close()
		os.Remove(fname)
	}

	return
}

// getMusicVideoID returns the ids for a specified title and artist
func getMusicVideoID(title string, artist string, expectedDuration ...int) (id string, err error) {
	searchTerm := strings.ToLower(strings.TrimSpace(title + " " + artist))
	youtubeSearchURL := fmt.Sprintf(
		`https://www.youtube.com/results?search_query=%s`,
		strings.Join(strings.Fields(searchTerm), "+"),
	)
	log.Debugf("searching url: %s", youtubeSearchURL)

	html, err := getPage(youtubeSearchURL)
	if err != nil {
		return
	}

	type Track struct {
		Title string
		ID    string
	}

	for _, line := range strings.Split(html, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `spell-correction-corrected`) && strings.Contains(line, `/results?`) && strings.Contains(line, `Search instead for`) {
			line = strings.Split(line, `Search instead for`)[1]
			youtubeSearchURL = "https://www.youtube.com" + getStringInBetween(line, `href="`, `"`)
			log.Debugf("getting new url: %s", youtubeSearchURL)
			html, err = getPage(youtubeSearchURL)
			if err != nil {
				return
			}
			break
		}
	}

	foundIDs := make(map[string]int)
	for _, line := range strings.Split(html, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `yt-lockup`) && strings.Contains(line, `/watch?v=`) {
			youtubeID := getStringInBetween(line, `/watch?v=`, `"`)
			youtubeID = strings.Split(youtubeID, "&amp;")[0]
			if _, ok := foundIDs[youtubeID]; ok {
				continue
			}
			if youtubeID == "" {
				continue
			}
			foundIDs[youtubeID] = len(foundIDs)
		}
	}

	type Job struct {
		Position int
		ID       string
	}
	type Result struct {
		Job         Job
		Rating      int
		YouTubeInfo YouTubeInfo
		Err         error
	}

	jobs := make(chan Job, len(foundIDs))
	results := make(chan Result, len(foundIDs))
	log.Debugf("processing %d found ids", len(foundIDs))
	for w := 0; w < len(foundIDs); w++ {
		go func(id int, jobs <-chan Job, results chan<- Result) {
			for j := range jobs {
				var errGet error
				var ytInfo YouTubeInfo
				ytInfo, errGet = getYoutubeVideoInfo(j.ID)
				if errGet != nil {
					results <- Result{
						Job: j,
						Err: err,
					}
					continue
				}

				descCheck := " " + strings.ToLower(ytInfo.Title) + " " + strings.Join(strings.Fields(strings.ToLower(ytInfo.Description)), " ") + " "
				log.Debug(descCheck)
				if !strings.Contains(descCheck, " "+strings.ToLower(title)+" ") {
					results <- Result{
						Job: j,
						Err: fmt.Errorf("no title found"),
					}
					continue
				}
				descCheck = strings.Replace(descCheck, " "+strings.ToLower(title)+" ", " ", -1)
				if !strings.Contains(descCheck, " "+strings.ToLower(artist)+" ") {
					results <- Result{
						Job: j,
						Err: fmt.Errorf("no artist found"),
					}
					continue
				}
				rating := 1
				if strings.Contains(descCheck, "provided to youtube") || strings.Contains(descCheck, "auto-generated by youtube") {
					rating = 2
				}
				results <- Result{
					Job:         j,
					Rating:      rating,
					YouTubeInfo: ytInfo,
				}
			}
		}(w, jobs, results)
	}

	for k := range foundIDs {
		jobs <- Job{
			Position: foundIDs[k],
			ID:       k,
		}
	}
	close(jobs)

	possibleVideos := make([]Result, len(foundIDs))
	for i := 0; i < len(foundIDs); i++ {
		result := <-results
		if result.Err != nil {
			log.Debugf("trying %s got error: %s", result.Job.ID, result.Err.Error())
		}
		possibleVideos[result.Job.Position] = result
	}

	var bestResult Result
	for i := range possibleVideos {
		if possibleVideos[i].Rating > bestResult.Rating {
			log.Debug("got one!")
			bestResult = possibleVideos[i]
		}
	}
	log.Debugf("best result: %+v", bestResult)
	if bestResult.Rating == 0 {
		err = fmt.Errorf("no id found")
	} else {
		id = bestResult.Job.ID
	}

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

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func getFfmpegBinary() (locationToBinary string, err error) {

	startTime := time.Now()
	defer func() {
		log.Debugf("time taken: %s", time.Since(startTime))
	}()
	cmd := exec.Command("ffmpeg", "-version")
	ffmpegOutput, errffmpeg := cmd.CombinedOutput()
	if errffmpeg == nil && strings.Contains(string(ffmpegOutput), "ffmpeg version") {
		locationToBinary = "ffmpeg"
		return
	}

	// if ffmpeg doesn't exist, then create it
	ffmpegFolder := path.Join(userHomeDir(), ".getsong")
	os.MkdirAll(ffmpegFolder, 0644)

	err = filepath.Walk(ffmpegFolder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			_, fname := filepath.Split(path)
			fname = strings.TrimRight(fname, filepath.Ext(fname))
			if fname == "ffmpeg" && (filepath.Ext(path) == ".exe" || filepath.Ext(path) == "") {
				locationToBinary = path
			}
			return nil
		})
	if err != nil {
		return
	}
	if locationToBinary != "" {
		return
	}

	urlToDownload := ""
	if runtime.GOOS == "windows" {
		urlToDownload = "https://ffmpeg.zeranoe.com/builds/win64/static/ffmpeg-4.1-win64-static.zip"
	} else {
		err = fmt.Errorf("Please install ffmpeg before continuing")
		return
	}

	var out io.Writer
	saveFile, err := os.Create(path.Join(ffmpegFolder, "ffmpeg.zip"))
	if err != nil {
		return
	}
	out = saveFile

	var req *http.Request
	req, err = http.NewRequest("GET", urlToDownload, nil)
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

	fmt.Println("Downloading ffmpeg...")
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
	saveFile.Close()
	if err != nil {
		return
	}

	_, err = unzip(path.Join(ffmpegFolder, "ffmpeg.zip"), ffmpegFolder)
	if err == nil {
		os.Remove(path.Join(ffmpegFolder, "ffmpeg.zip"))
	}
	return
}

// unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}

type YouTubeInfo struct {
	Title       string
	Description string
	ID          string
}

func getYoutubeVideoInfo(id string) (ytInfo YouTubeInfo, err error) {
	youtubeSearchURL := fmt.Sprintf(
		`https://www.youtube.com/watch?v=%s`,
		id,
	)
	log.Debugf("getting ytinfo for url: %s", youtubeSearchURL)

	html, err := getPage(youtubeSearchURL)
	if err != nil {
		return
	}

	ytInfo.ID = id
	for _, line := range strings.Split(html, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `meta property="og:title"`) {
			ytInfo.Title = getStringInBetween(line, `content="`, `"`)
		} else if strings.Contains(line, `meta property="og:description"`) {
			ytInfo.Description = getStringInBetween(line, `content="`, `"`)
			return
		}
	}
	err = fmt.Errorf("could not find info")
	return
}

func getPage(urlToGet string) (html string, err error) {
	var client http.Client
	resp, err := client.Get(urlToGet)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			err = err2
			return
		}
		html = string(bodyBytes)
	} else {
		err = fmt.Errorf("could not get page")
	}
	return
}

// deprecated
// var getpagejs = []byte(`const puppeteer = require('puppeteer');

// (async() => {
// 	const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox'],headless:true});
// 	const page = await browser.newPage();
// 	await page.goto(process.argv[2]);
//     await page.waitFor(1000);
// 	let content = await page.content();
// 	console.log(content);
// 	browser.close();
// })();`)

// func getRenderedPage(urlToGet string) (html string, err error) {
// 	html, err = getRenderedPageUsingNode(urlToGet)
// 	if err != nil {
// 		// get page using server
// 		html, err = getRenderedPageUsingServer(urlToGet)
// 	}
// 	return
// }

// func getRenderedPageUsingServer(urlToGet string) (html string, err error) {
// 	log.Debug("getting rendered page from server")
// 	var client http.Client
// 	resp, err := client.Get("https://getrenderedpage.schollz.com/" + urlToGet)
// 	if err != nil {
// 		return
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode == http.StatusOK {
// 		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
// 		if err2 != nil {
// 			err = err2
// 			return
// 		}
// 		html = string(bodyBytes)
// 	} else {
// 		err = fmt.Errorf("could not get from server")
// 	}
// 	return
// }

// func getRenderedPageUsingNode(urlToGet string) (html string, err error) {
// 	tmpfile, err := ioutil.TempFile(".", "getpage.*.js")
// 	if err != nil {
// 		return "", err
// 	}

// 	defer os.Remove(tmpfile.Name()) // clean up

// 	if _, err = tmpfile.Write(getpagejs); err != nil {
// 		return "", err
// 	}
// 	if err = tmpfile.Close(); err != nil {
// 		return "", err
// 	}

// 	log.Debugf("%s %s %s", "node", tmpfile.Name(), urlToGet)
// 	cmd := exec.Command("node", tmpfile.Name(), urlToGet)
// 	var htmlBytes []byte
// 	htmlBytes, err = cmd.CombinedOutput()
// 	html = string(htmlBytes)
// 	return
// }
