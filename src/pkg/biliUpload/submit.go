package biliUpload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

//test
type submitParams struct {
	Copyright    int    `json:"copyright"`
	Source       string `json:"source"`
	Tid          int    `json:"tid"`
	Cover        string `json:"cover"`
	Title        string `json:"title"`
	DescFormatId int    `json:"desc_format_id"`
	Desc         string `json:"desc"`
	Dynamic      string `json:"dynamic"`
	Subtitle     struct {
		Open int    `json:"open"`
		Lan  string `json:"lan"`
	} `json:"subtitle"`
	Videos []uploadRes `json:"videos"`
	Tags   string      `json:"tag"`
	Dtime  int         `json:"dtime"`
}

func submit(u Biliup, v []*uploadRes) error {
	if u.Title == "" {
		u.Title = v[0].Title
	}
	params := submitParams{
		Copyright:    u.Copyright,
		Source:       u.Source,
		Tid:          u.Tid,
		Cover:        u.Cover,
		Title:        u.Title,
		DescFormatId: 0,
		Desc:         u.Description,
		Dynamic:      "",
		Tags:         strings.Join(u.Tag, ","),
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
	for i := 0; i <= 20; i++ {
		req, _ := http.NewRequest("POST", "http://member.bilibili.com/x/vu/client/add?access_key="+u.User.AccessToken, bytes.NewBuffer(params_str))
		req.Header = Header
		res, err := client.Do(req)
		if err != nil {
			fmt.Println("提交出现问题", err.Error())
			if i == 20 {
				return err
			}
			continue
		}
		body, _ := ioutil.ReadAll(res.Body)
		t := struct {
			Code int `json:"code"`
		}{}
		_ = json.Unmarshal(body, &t)
		if t.Code != 0 {
			fmt.Println("提交出现问题", string(body))
			if i == 20 {
				return errors.New("提交出现问题")
			}
		} else {
			break
		}
		res.Body.Close()
	}

	return nil
}
