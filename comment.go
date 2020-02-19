package main

import (
	dbstorage "github.com/nektro/go.dbstorage"
	"github.com/valyala/fastjson"
)

type tComment struct {
	ID        int64 `json:"id"`
	IDS       string
	Subreddit string `json:"subreddit" sqlite:"text"`
	CommentID string `json:"comment_id" sqlite:"text"`
	ParentID  string `json:"parent_id" sqlite:"text"`
	PostID    string `json:"post_id" sqlite:"text"`
	Json      string `json:"json" sqlite:"text"`
	Author    string `json:"author" sqlite:"text"`
	Submitted int64  `json:"submitted_at" sqlite:"int"`
	Body      string `json:"body" sqlite:"text"`
}

func insertComment(sb, ci, ri, pi, js, au string, sm int64, bd string) {
	dbC.Build().Ins("comments").Lock()
	id := dbC.QueryNextID("comments")
	dbC.QueryPrepared(true, "insert into comments values (?,?,?,?,?,?,?,?,?)", id, sb, ci, ri, pi, js, au, sm, bd)
	dbC.Build().Ins("comments").Unlock()
}

func doesCommentExist(id string) bool {
	return dbstorage.QueryHasRows(dbC.Build().Se("*").Fr("comments").Wh("comment_id", id).Exe())
}

func commentListingCb(t, name string, item *fastjson.Value) (bool, bool) {
	if !doComms {
		return true, true
	}

	idC := string(item.GetStringBytes("data", "name"))

	if doesCommentExist(idC) {
		return true, true
	}

	idR := string(item.GetStringBytes("data", "parent_id"))
	idP := string(item.GetStringBytes("data", "link_id"))
	jsn := string(item.MarshalTo([]byte{}))
	ath := string(item.GetStringBytes("data", "author"))
	sbm := int64(item.GetFloat64("data", "created"))
	body := string(item.GetStringBytes("data", "body"))

	insertComment(name, idC, idR, idP, jsn, ath, sbm, body)

	return false, false
}
