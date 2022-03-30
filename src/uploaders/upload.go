package uploaders

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
)

type uploadRes struct {
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Desc     string `json:"desc"`
}
type User struct {
	SESSDATA          string
	bili_jct          string
	DedeUserID        string
	DedeUserID__ckMd5 string
	access_token      string
}

type UploadedVideoInfo struct {
	title    string
	filename string
	desc     string
}

var client http.Client

//临时logger

func init() {

	//临时logger

	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Printf("Got error while creating cookie jar %s", err.Error())
	}
	client = http.Client{
		Jar: jar,
	}
}

func cookie_login_check(u User) error {
	cookie := []*http.Cookie{{Name: "SESSDATA", Value: u.SESSDATA},
		{Name: "DedeUserID", Value: u.DedeUserID},
		{Name: "DedeUserID__ckMd5", Value: u.DedeUserID__ckMd5},
		{Name: "bili_jct", Value: u.bili_jct}}
	urlObj, _ := url.Parse("https://api.bilibili.com")
	client.Jar.SetCookies(urlObj, cookie)
	apiUrl := "https://api.bilibili.com/x/web-interface/nav"
	req, _ := http.NewRequest("GET", apiUrl, nil)
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	var t struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(body, &t)
	if t.Code != 0 {
		return errors.New("cookie login failed")
	}
	urlObj, _ = url.Parse("https://member.bilibili.com")
	client.Jar.SetCookies(urlObj, cookie)
	return nil
}
func upload(file *os.File, user User) (*uploadRes, error) {
	if err := cookie_login_check(user); err != nil {
		fmt.Println("cookie 校验失败")
		return &uploadRes{}, err
	}
	state, _ := file.Stat()
	q := struct {
		R       string `url:"r"`
		Profile string `url:"profile"`
		Ssl     int    `url:"ssl"`
		Version string `url:"version"`
		Build   int    `url:"build"`
		Name    string `url:"name"`
		Size    int    `url:"size"`
	}{
		R:       "upos",
		Profile: "ugcupos/bup",
		Ssl:     0,
		Version: "2.8.1.2",
		Build:   2081200,
		Name:    file.Name(),
		Size:    int(state.Size()),
	}
	v, _ := query.Values(q)
	queryUrl := "https://member.bilibili.com/preupload?upcdn=ws&probe_version=20200810"
	req, _ := http.NewRequest("GET", queryUrl+v.Encode(), nil)
	res, _ := client.Do(req)
	var body upos_upload_segments
	content, _ := ioutil.ReadAll(res.Body)
	_ = json.Unmarshal(content, &body)
	if body.Ok != 1 {
		return &uploadRes{}, errors.New("query upload failed")
	}
	videoInfo, err := upos(file, int(state.Size()), body)
	return videoInfo, err
}
