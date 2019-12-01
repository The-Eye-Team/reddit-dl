package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nektro/go-util/ansi/style"
	"github.com/nektro/go-util/mbpp"
	"github.com/nektro/go-util/util"
	"github.com/spf13/pflag"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"
)

var (
	DoneDir = "./data/"
	logF    *os.File
)
var (
	netClient = &http.Client{
		Timeout: time.Second * 10,
	}
)

func main() {
	flagSubr := pflag.StringArrayP("subreddit", "r", []string{}, "")
	flagUser := pflag.StringArrayP("user", "u", []string{}, "")

	flagSaveDir := pflag.String("save-dir", "", "")
	flagConcurr := pflag.Int("concurrency", 30, "")

	pflag.Parse()

	//

	if len(*flagSaveDir) > 0 {
		DoneDir = *flagSaveDir
	}
	DoneDir, _ = filepath.Abs(DoneDir)
	DoneDir += "/reddit.com"

	logF, _ = os.Create("./log.txt")

	util.RunOnClose(onClose)
	log.SetOutput(logF)
	mbpp.Init(*flagConcurr)

	//

	rb := mbpp.CreateHeadlessJob("reddit.com subreddits", int64(len(*flagSubr)), nil)
	for _, item := range *flagSubr {
		fetchListing("r", item, "")
		rb.Increment(1)
	}
	rb.Done()

	ub := mbpp.CreateHeadlessJob("reddit.com users", int64(len(*flagUser)), nil)
	for _, item := range *flagUser {
		fetchListing("u", item, "")
		ub.Increment(1)
	}
	ub.Done()

	//

	time.Sleep(time.Second / 2)
	mbpp.Wait()
	onClose()
}

func onClose() {
	util.Log("Download complete after", (mbpp.GetTaskCount()), "jobs and", util.ByteCountIEC(mbpp.GetTaskDownloadSize()), "downloaded.")
	logF.Close()
}

func fetchListing(t, name, after string) {
	next := ""

	jobname := style.FgRed + t + style.ResetFgColor + "/"
	jobname += style.FgCyan + name + style.ResetFgColor + " +"
	jobname += style.FgYellow + after + style.ResetFgColor
	mbpp.CreateJob(jobname, func(bar1 *mbpp.BarProxy) {
		if len(after) > 0 {
			after = "&after=" + after
		}
		res, _ := nRequest(http.MethodGet, "https://old.reddit.com/"+t+"/"+name+"/.json?show=all"+after)
		bys, _ := ioutil.ReadAll(res.Body)
		val, _ := fastjson.Parse(string(bys))

		//
		dir := DoneDir + "/" + t + "/" + name
		os.MkdirAll(dir, os.ModePerm)

		next = string(val.GetStringBytes("data", "after"))

		ar := val.GetArray("data", "children")
		bar1.AddToTotal(int64(len(ar)))
		for _, item := range ar {
			id := string(item.GetStringBytes("data", "id"))
			title := string(item.GetStringBytes("data", "title"))
			urlS := string(item.GetStringBytes("data", "url"))
			selftext := string(item.GetStringBytes("data", "selftext"))
			selftexth := string(item.GetStringBytes("data", "selftext_html"))
			jtem := item

			//

			dir2 := dir + "/" + id[:2] + "/" + id
			if util.DoesDirectoryExist(dir2) {
				next = ""
				return
			}
			os.MkdirAll(dir2, os.ModePerm)

			go saveTextToJob(F("%s/%s/%s api_data.json", t, name, id), dir2+"/api_data.json", string(jtem.MarshalTo([]byte{})))

			//
			st := id + "\n" + urlS + "\n" + title + "\n\n"
			go saveTextToJob(F("%s/%s/%s selftext.txt", t, name, id), dir2+"/selftext.txt", st+selftext)
			go saveTextToJob(F("%s/%s/%s selftext.html", t, name, id), dir2+"/selftext.html", selftexth)

			//
			urlO, err := url.Parse(urlS)
			if err != nil {
				//
				fmt.Fprintln(logF, "error:", 1, t, name, id, urlS)
				continue
			}

			res, err := netClient.Head(urlS)
			if err != nil {
				fmt.Fprintln(logF, "error:", 2, t, name, id, urlS)
				return
			}
			ct := res.Header.Get("content-type")
			l := true

			if urlO.Host == "old.reddit.com" {
				l = false
			}
			if urlO.Host == "i.redd.it" || urlO.Host == "i.imgur.com" || (urlO.Host == "imgur.com" && !strings.Contains(ct, "text/html")) {
				go mbpp.CreateDownloadJob(urlS, dir2+"/"+urlO.Host+"_"+urlO.Path[1:], nil)
				l = false
			}
			if urlO.Host == "imgur.com" && strings.Contains(ct, "text/html") {
				res, _ := nRequest(http.MethodGet, urlS)
				doc, _ := goquery.NewDocumentFromResponse(res)
				doc.Find(".post-images .post-image-container").Each(func(_ int, el *goquery.Selection) {
					pid, _ := el.Attr("id")
					ext := findExtension("https://i.imgur.com/" + pid + ".png")
					go mbpp.CreateDownloadJob("https://i.imgur.com/"+pid+ext, dir2+"/"+urlO.Host+"_"+pid+ext, nil)
				})
				l = false
			}
			if urlO.Host == "media.giphy.com" && ct == "image/gif" {
				pid := strings.Split(urlS, "/")[2]
				go mbpp.CreateDownloadJob(urlS, dir2+"/"+urlO.Host+"_"+pid+".gif", nil)
				l = false
			}
			if strings.Contains(ct, "text/html") {
				l = false

			} else {
				fn := strings.TrimPrefix(urlO.Path, filepath.Dir(urlO.Path))
				go mbpp.CreateDownloadJob(urlS, dir2+"/"+urlO.Host+"_"+fn, nil)
				l = false

			}
			if l {
				fmt.Fprintln(logF, t, name, id, urlO.Host, ct, urlS)
			}

			bar1.Increment(1)
		}
	})
	if len(next) > 0 {
		fetchListing(t, name, next)
	}
}

func saveTextToJob(name, path, text string) {
	if util.DoesFileExist(path) {
		return
	}
	to, err := os.Create(path)
	if err != nil {
		return
	}
	defer to.Close()
	fmt.Fprintln(to, text)
}

func nRequest(method, urlS string) (*http.Response, error) {
	req, err := http.NewRequest(method, urlS, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("user-agent", "github.com/nektro")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func findExtension(urlS string) string {
	res, _ := nRequest(http.MethodHead, urlS)
	ext, _ := mime.ExtensionsByType(res.Header.Get("content-type"))
	return ext[0]
}

func TrimLen(s string, max int) string {
	return s[:Min(len(s), max)]
}

func Min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
