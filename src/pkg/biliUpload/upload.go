package biliUpload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"path/filepath"
	"time"
)

var ChunkSize int = 10485760

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
type UploadRes struct {
	Title    string      `json:"title"`
	Filename string      `json:"filename"`
	Desc     string      `json:"desc"`
	Info     interface{} `json:"-"`
}

type UploadedVideoInfo struct {
	title    string
	filename string
	desc     string
}
type uploadOs struct {
	os       string
	query    string
	probeUrl string
}
type UploadedFile struct {
	FilePath string
	FileName string
}

var client http.Client
var Header = http.Header{
	"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/63.0.3239.108"},
	"Referer":    []string{"https://www.bilibili.com"},
	"Connection": []string{"keep-alive"},
}

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
func selectUploadOs(lines string) uploadOs {
	var os uploadOs
	if lines == "auto" {
	} else {
		if lines == "bda2" {
			os = uploadOs{
				os:       "upos",
				query:    "upcdn=bda2&probe_version=20200810",
				probeUrl: "//upos-sz-upcdnbda2.bilivideo.com/OK",
			}
		} else if lines == "ws" {
			os = uploadOs{
				os:       "upos",
				query:    "upcdn=ws&probe_version=20200810",
				probeUrl: "//upos-sz-upcdnws.bilivideo.com/OK",
			}
		} else if lines == "qn" {
			os = uploadOs{
				os:       "upos",
				query:    "upcdn=qn&probe_version=20200810",
				probeUrl: "//upos-sz-upcdnqn.bilivideo.com/OK",
			}
		} else if lines == "cos" {
			os = uploadOs{
				os:       "cos",
				query:    "",
				probeUrl: "",
			}
		} else if lines == "cos-internal" {
			os = uploadOs{
				os:       "cos-internal",
				query:    "",
				probeUrl: "",
			}
		}
	}
	return os
}
func UploadFile(file *os.File, user User, lines string) (*UploadRes, error) {
	if err := CookieLoginCheck(user); err != nil {
		fmt.Println("cookie 校验失败")
		return &UploadRes{}, err
	}
	upOs := selectUploadOs(lines)
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
		Ssl:     0,
		Version: "2.8.1.2",
		Build:   2081200,
		Name:    filepath.Base(file.Name()),
		Size:    int(state.Size()),
	}
	if upOs.os == "cos-internal" {
		q.R = "cos"
	} else {
		q.R = upOs.os
	}
	if upOs.os == "upos" {
		q.Profile = "ugcupos/bup"
	} else {
		q.Profile = "ugcupos/bupfetch"
	}
	v, _ := query.Values(q)
	client.Timeout = time.Second * 5
	req, _ := http.NewRequest("GET", "https://member.bilibili.com/preupload?"+upOs.query+v.Encode(), nil)
	res, _ := client.Do(req)
	defer res.Body.Close()
	content, _ := ioutil.ReadAll(res.Body)
	if upOs.os == "cos-internal" || upOs.os == "cos" {
		var internal bool
		if upOs.os == "cos-internal" {
			internal = true
		}
		body := &cosUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			return &UploadRes{}, errors.New("query Upload Parameters failed")
		}
		videoInfo, err := cos(file, int(state.Size()), body, internal, ChunkSize)
		return videoInfo, err

	} else if upOs.os == "upos" {
		body := &uposUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			return &UploadRes{}, errors.New("query UploadFile failed")
		}
		videoInfo, err := upos(file, int(state.Size()), body)
		return videoInfo, err
	}
	return &UploadRes{}, errors.New("unknown upload os")
}
func FolderUpload(folder string, u User, lines string) ([]*UploadRes, []UploadedFile, error) {
	dir, err := ioutil.ReadDir(folder)
	if err != nil {
		fmt.Printf("read dir error:%s", err)
		return nil, nil, err
	}
	var uploadedFiles []UploadedFile
	var submitFiles []*UploadRes
	for _, file := range dir {
		filename := filepath.Join(folder, file.Name())
		//now := time.Now()
		//if diff := now.Sub(file.ModTime()); diff.Minutes() < 3 {
		//	fmt.Printf("%s is too new, skip it\n", filename)
		//	continue
		//}
		uploadFile, err := os.Open(filename)
		if err != nil {
			log.Printf("open file %s error:%s", filename, err)
			continue
		}
		videoPart, err := UploadFile(uploadFile, u, lines)
		if err != nil {
			log.Printf("UploadFile file error:%s", err)
			uploadFile.Close()
			continue
		}
		uploadedFiles = append(uploadedFiles, UploadedFile{
			FilePath: folder,
			FileName: file.Name(),
		})
		submitFiles = append(submitFiles, videoPart)
		uploadFile.Close()
	}
	return submitFiles, uploadedFiles, nil
}
func UploadFolderWithSubmit(uploadPath string, Biliup Biliup) ([]UploadedFile, error) {
	var submitFiles []*UploadRes
	if !filepath.IsAbs(uploadPath) {
		pwd, _ := os.Getwd()
		uploadPath = filepath.Join(pwd, uploadPath)
	}
	fmt.Println(uploadPath)
	submitFiles, uploadedFile, err := FolderUpload(uploadPath, Biliup.User, Biliup.UploadLines)
	if err != nil {
		fmt.Printf("UploadFile file error:%s", err)
		return nil, err
	}
	err = Submit(Biliup, submitFiles)
	if err != nil {
		fmt.Printf("Submit file error:%s", err)
		return nil, err
	}
	return uploadedFile, nil
}
