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
	dbstorage "github.com/nektro/go.dbstorage"
	"github.com/spf13/pflag"
	"github.com/valyala/fastjson"
)

var (
	DoneDir = "./data/"
	logF    *os.File
	db      dbstorage.Database
)
var (
	netClient = &http.Client{
		Timeout: time.Second * 5,
	}
)

func main() {
	flagSubr := pflag.StringArrayP("subreddit", "r", []string{}, "The name of a subreddit to archive. (ex. AskReddit, unixporn, CasualConversation, etc.)")
	flagUser := pflag.StringArrayP("user", "u", []string{}, "The name of a subreddit to archive. (ex. spez, PoppinKREAM, Shitty_Watercolour, etc.)")

	flagSaveDir := pflag.String("save-dir", "", "Path to a directory to save to.")
	flagConcurr := pflag.Int("concurrency", 10, "Maximum number of simultaneous downloads.")

	pflag.Parse()

	//

	if len(*flagSaveDir) > 0 {
		DoneDir = *flagSaveDir
	}
	DoneDir, _ = filepath.Abs(DoneDir)
	DoneDir += "/reddit.com"
	os.MkdirAll(DoneDir, os.ModePerm)

	logF, _ = os.Create(DoneDir + "/log.txt")

	db = dbstorage.ConnectSqlite(DoneDir + "/posts.db")
	db.CreateTableStruct("posts", Post{})

	//

	util.RunOnClose(onClose)
	log.SetOutput(logF)
	mbpp.Init(*flagConcurr)
	mbpp.SetBarStyle("|-ᗧ•ᗣ")

	//

	rb := mbpp.CreateHeadlessJob("reddit.com subreddits", int64(len(*flagSubr)), nil)
	for _, item := range *flagSubr {
		fetchListing("r/"+item, "")
		rb.Increment(1)
	}
	rb.Done()

	ub := mbpp.CreateHeadlessJob("reddit.com users", int64(len(*flagUser)), nil)
	for _, item := range *flagUser {
		fetchListing("u/"+item+"/submitted", "")
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

func fetchListing(loc, after string) {
	next := ""
	s := strings.Split(loc, "/")
	t := s[0]
	name := s[1]

	jobname := style.FgRed + t + style.ResetFgColor + "/"
	jobname += style.FgCyan + name + style.ResetFgColor + " +"
	jobname += style.FgYellow + after + style.ResetFgColor
	mbpp.CreateJob(jobname, func(bar1 *mbpp.BarProxy) {
		if len(after) > 0 {
			after = "&after=" + after
		}
		res, _ := fetch(http.MethodGet, "https://old.reddit.com/"+loc+"/.json?show=all"+after)
		bys, _ := ioutil.ReadAll(res.Body)
		val, _ := fastjson.Parse(string(bys))

		next = string(val.GetStringBytes("data", "after"))

		ar := val.GetArray("data", "children")
		bar1.AddToTotal(int64(len(ar)))
		for _, item := range ar {
			id := string(item.GetStringBytes("data", "id"))
			jtem := item

			//
			if DoesPostExist(id) {
				next = ""
				bar1.Increment(1)
				continue
			}

			title := string(item.GetStringBytes("data", "title"))
			pjson := string(jtem.MarshalTo([]byte{}))
			urlS := string(item.GetStringBytes("data", "url"))
			sub := string(item.GetStringBytes("data", "subreddit"))
			author := string(item.GetStringBytes("data", "author"))
			InsertPost(sub, id, title, pjson, urlS, author)

			dir := DoneDir + "/r/" + sub
			dir2 := dir + "/" + id[:2] + "/" + id
			if util.DoesDirectoryExist(dir2) {
				bar1.Increment(1)
				continue
			}

			downloadPost(t, name, id, urlS, dir2)

			bar1.Increment(1)
		}
	})
	if len(next) > 0 {
		fetchListing(loc, next)
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

func fetch(method, urlS string) (*http.Response, error) {
	req, _ := http.NewRequest(method, urlS, nil)
	req.Header.Add("user-agent", "the-eye-team/reddit-dl")
	res, _ := http.DefaultClient.Do(req)
	return res, nil
}

func findExtension(urlS string) string {
	res, _ := fetch(http.MethodHead, urlS)
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

func downloadPost(t, name string, id string, urlS string, dir string) {
	urlO, err := url.Parse(urlS)
	if err != nil {
		//
		fmt.Fprintln(logF, "error:", 1, t, name, id, urlS)
		return
	}

	res, err := netClient.Head(urlS)
	if err != nil {
		fmt.Fprintln(logF, "error:", 2, t, name, id, urlS)
		return
	}

	links := [][2]string{}
	ct := res.Header.Get("content-type")
	l := true

	if urlO.Host == "old.reddit.com" {
		return
	}
	if urlO.Host == "i.redd.it" || urlO.Host == "i.imgur.com" || (urlO.Host == "imgur.com" && !strings.Contains(ct, "text/html")) {
		links = append(links, [2]string{urlS, urlO.Host + "_" + urlO.Path[1:]})
		l = false
	}
	if urlO.Host == "imgur.com" && strings.Contains(ct, "text/html") {
		res, _ := fetch(http.MethodGet, urlS)
		doc, _ := goquery.NewDocumentFromResponse(res)
		doc.Find(".post-images .post-image-container").Each(func(_ int, el *goquery.Selection) {
			pid, _ := el.Attr("id")
			ext := findExtension("https://i.imgur.com/" + pid + ".png")
			links = append(links, [2]string{"https://i.imgur.com/" + pid + ext, urlO.Host + "_" + pid + ext})
		})
		l = false
	}
	if urlO.Host == "media.giphy.com" && ct == "image/gif" {
		pid := strings.Split(urlS, "/")[2]
		links = append(links, [2]string{urlS, urlO.Host + "_" + pid + ".gif"})
		l = false
	}
	if strings.Contains(ct, "text/html") {
		l = false

	} else {
		fn := strings.TrimPrefix(urlO.Path, filepath.Dir(urlO.Path))
		links = append(links, [2]string{urlS, urlO.Host + "_" + fn})
		l = false

	}
	if l {
		fmt.Fprintln(logF, t, name, id, urlO.Host, ct, urlS)
	}
	if len(links) > 0 {
		os.MkdirAll(dir, os.ModePerm)

		for _, item := range links {
			go mbpp.CreateDownloadJob(item[0], dir+"/"+item[1], nil)
		}
	}
}
