package main

import (
	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"
	"github.com/valyala/fastjson"
)

type tPost struct {
	ID        int64 `json:"id"`
	IDS       string
	Subreddit string `json:"subreddit" sqlite:"text"`
	PostID    string `json:"post_id" sqlite:"text"`
	Title     string `json:"title" sqlite:"text"`
	PostJson  string `json:"json" sqlite:"text"`
	Link      string `json:"link" sqlite:"text"`
	Author    string `json:"author" sqlite:"text"`
	Submitted int64  `json:"submitted_at" sqlite:"int"`
}

func insertPost(sub, post, title, pjson, link, author string, submittedAt int64) {
	dbP.Build().Ins("posts").Lock()
	id := dbP.QueryNextID("posts")
	dbP.QueryPrepared(true, "insert into posts values (?, ?, ?, ?, ?, ?, ?, ?)", id, sub, post, title, pjson, link, author, submittedAt)
	dbP.Build().Ins("posts").Unlock()
}

func doesPostExist(post string) bool {
	return dbstorage.QueryHasRows(dbP.Build().Se("*").Fr("posts").Wh("post_id", post).Exe())
}

func postListingCb(t, name string, item *fastjson.Value) (bool, bool) {
	id := string(item.GetStringBytes("data", "id"))

	//
	if doesPostExist(id) {
		return true, true
	}

	title := string(item.GetStringBytes("data", "title"))
	pjson := string(item.MarshalTo([]byte{}))
	urlS := string(item.GetStringBytes("data", "url"))
	sub := string(item.GetStringBytes("data", "subreddit"))
	author := string(item.GetStringBytes("data", "author"))
	postedAt := int64(item.GetFloat64("data", "created_utc"))
	insertPost(sub, id, title, pjson, urlS, author, postedAt)

	dir := DoneDir + "/r/" + sub
	dir2 := dir + "/" + id[:2] + "/" + id
	if util.DoesDirectoryExist(dir2) {
		return false, true
	}

	go downloadPost(t, name, id, urlS, dir2)

	return false, false
}
