package biliUpload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
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
	Videos []UploadRes `json:"videos"`
	Tags   string      `json:"tag"`
	Dtime  int         `json:"dtime"`
}

func VerifyAndFix(params *submitParams) error {
	if params.Copyright < 1 || params.Copyright > 2 {
		params.Copyright = 2
		return errors.New("copyright must be 1 or 2,Set to 2")
	}
	if params.Copyright == 2 && params.Source == "" {
		params.Source = "转载地址"
		return errors.New("when copyright is 2,source must be set")
	}
	if params.Tid <= 0 {
		params.Tid = 122
		return errors.New("tid must be set,Set to 122")
	}
	if params.Title == "" {
		params.Title = "标题"
		return errors.New("title must not be empty,set to '标题'")
	}
	return nil
}
func Submit(u Biliup, v []*UploadRes) error {
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
	err := VerifyAndFix(&params)
	if err != nil {
		log.Println(err)
	}
	for i := range v {
		params.Videos = append(params.Videos, *v[i])
	}
	paramsStr, _ := json.Marshal(params)
	for i := 0; i <= 20; i++ {
		time.Sleep(time.Second * 5)
		req, _ := http.NewRequest("POST", "http://member.bilibili.com/x/vu/client/add?access_key="+u.User.AccessToken, bytes.NewBuffer(paramsStr))
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
