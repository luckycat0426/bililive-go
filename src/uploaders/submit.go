package uploaders

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Video_infos struct {
	tid         string
	title       string
	tags        []string
	source      string
	cover       string
	cover_path  string
	description string
	copyright   int
}
type submit_params struct {
	Copyright      int    `json:"copyright"`
	Source         string `json:"source"`
	Tid            string `json:"tid"`
	Cover          string `json:"cover"`
	Title          string `json:"title"`
	Desc_format_id int    `json:"desc_format_id"`
	Desc           string `json:"desc"`
	Dynamic        string `json:"dynamic"`
	Subtitle       struct {
		Open int    `json:"open"`
		Lan  string `json:"lan"`
	} `json:"subtitle"`
	Videos []uploadRes `json:"videos"`
	Tags   string      `json:"tags"`
	Dtime  int         `json:"dtime"`
}

func submit(u Biliup, v []*uploadRes) error {
	if u.title == "" {
		u.title = v[0].Title
	}
	params := submit_params{
		Copyright:      u.copyright,
		Source:         u.source,
		Tid:            u.tid,
		Cover:          u.cover,
		Title:          u.title,
		Desc_format_id: 0,
		Desc:           u.description,
		Dynamic:        "",
		Tags:           strings.Join(u.tags, ","),
		Subtitle: struct {
			Open int    `json:"open"`
			Lan  string `json:"lan"`
		}{
			Open: 0,
			Lan:  "",
		},
		Dtime: 0,
	}
	for i := range v {
		params.Videos = append(params.Videos, *v[i])
	}
	params_str, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "http://member.bilibili.com/x/vu/client/add?access_key="+u.user.access_token, bytes.NewBuffer(params_str))
	req.Header = Header
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("提交出现问题", err.Error())
		return err
	}
	body, _ := ioutil.ReadAll(res.Body)
	t := struct {
		Code int `json:"code"`
	}{}
	_ = json.Unmarshal(body, &t)
	if t.Code != 0 {
		fmt.Println("提交出现问题", string(body))
		return errors.New("提交出现问题")
	}
	return nil
}
