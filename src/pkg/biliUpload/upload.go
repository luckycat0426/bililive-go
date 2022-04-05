package biliUpload

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

	"github.com/pkg/profile"

	"path/filepath"
	"time"
)

type Biliup struct {
	User        User   `json:"user"`
	Lives       string `json:"url"`
	UploadLines string `json:"upload_lines"`
	Threads     int    `json:"threads"`
	VideoInfos
}
type VideoInfos struct {
	Tid         int      `json:"tid"`
	Title       string   `json:"title"`
	Tag         []string `json:"tag,omitempty"`
	Source      string   `json:"source,omitempty"`
	Cover       string   `json:"cover,omitempty"`
	CoverPath   string   `json:"cover_path,omitempty"`
	Description string   `json:"description,omitempty"`
	Copyright   int      `json:"copyright,omitempty"`
}
type User struct {
	SESSDATA        string `json:"SESSDATA"`
	BiliJct         string `json:"bili_jct"`
	DedeUserID      string `json:"DedeUserID"`
	DedeuseridCkmd5 string `json:"DedeUserID__ckMd5"`
	AccessToken     string `json:"access_token"`
}
type uploadRes struct {
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Desc     string `json:"desc"`
}

type UploadedVideoInfo struct {
	title    string
	filename string
	desc     string
}

var client http.Client

func init() {

	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Printf("Got error while creating cookie jar %s", err.Error())
	}
	client = http.Client{
		Jar: jar,
	}
}

func CookieLoginCheck(u User) error {
	cookie := []*http.Cookie{{Name: "SESSDATA", Value: u.SESSDATA},
		{Name: "DedeUserID", Value: u.DedeUserID},
		{Name: "DedeUserID__ckMd5", Value: u.DedeuseridCkmd5},
		{Name: "bili_jct", Value: u.BiliJct}}
	urlObj, _ := url.Parse("https://api.bilibili.com")
	client.Jar.SetCookies(urlObj, cookie)
	apiUrl := "https://api.bilibili.com/x/web-interface/nav"
	req, _ := http.NewRequest("GET", apiUrl, nil)
	res, _ := client.Do(req)
	defer res.Body.Close()
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
	if err := CookieLoginCheck(user); err != nil {
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
		Name:    filepath.Base(file.Name()),
		Size:    int(state.Size()),
	}
	v, _ := query.Values(q)
	queryUrl := "https://member.bilibili.com/preupload?upcdn=ws&probe_version=20200810"
	req, _ := http.NewRequest("GET", queryUrl+v.Encode(), nil)
	res, _ := client.Do(req)
	defer res.Body.Close()
	var body upos_upload_segments
	content, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(content))
	_ = json.Unmarshal(content, &body)
	if body.Ok != 1 {
		return &uploadRes{}, errors.New("query upload failed")
	}
	videoInfo, err := upos(file, int(state.Size()), body)
	return videoInfo, err
}
func FolderUpload(folder string, u User) ([]*uploadRes, error) {
	dir, err := ioutil.ReadDir(folder)
	if err != nil {
		fmt.Printf("read dir error:%s", err)
		return nil, err
	}
	var submitFiles []*uploadRes
	for _, file := range dir {
		filename := filepath.Join(folder, file.Name())
		now := time.Now()
		fmt.Println(file.ModTime())
		fmt.Println(now.Sub(file.ModTime()))
		if diff := now.Sub(file.ModTime()); diff.Minutes() < 3 {
			fmt.Printf("%s is too new, skip it\n", filename)
			continue
		}
		uploadFile, err := os.Open(filename)
		if err != nil {
			fmt.Printf("open file error:%s", err)
			return nil, err
		}
		videoPart, err := upload(uploadFile, u)
		if err != nil {
			fmt.Printf("upload file error:%s", err)
			uploadFile.Close()
			continue
		}
		submitFiles = append(submitFiles, videoPart)
		uploadFile.Close()
	}
	return submitFiles, nil
}
func MainUpload(uploadPath string, Biliup Biliup) error {
	defer profile.Start().Stop()
	var submitFiles []*uploadRes
	if !filepath.IsAbs(uploadPath) {
		pwd, _ := os.Getwd()
		uploadPath = filepath.Join(pwd, uploadPath)
	}
	fmt.Println(uploadPath)
	submitFiles, err := FolderUpload(uploadPath, Biliup.User)
	if err != nil {
		fmt.Printf("upload file error:%s", err)
		return err
	}
	err = submit(Biliup, submitFiles)
	if err != nil {
		fmt.Printf("submit file error:%s", err)
		return err
	}
	return nil
}
