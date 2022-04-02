package biliUpload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-querystring/query"
)

const Threads = 4

var Header = http.Header{
	"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/63.0.3239.108"},
	"Referer":    []string{"https://www.bilibili.com"},
	"Connection": []string{"keep-alive"},
}

type upos_upload_segments struct {
	Ok        int    `json:"OK"`
	Auth      string `json:"auth"`
	BizID     int    `json:"biz_id"`
	ChunkSize int    `json:"chunk_size"`
	Endpoint  string `json:"endpoint"`
	Uip       string `json:"uip"`
	UposURI   string `json:"upos_uri"`
}
type pre_upload_json struct {
	UploadID string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Ok       int    `json:"OK"`
	Key      string `json:"key"`
}
type upload_param struct {
	Name     string `url:"name"`
	UploadId string `url:"uploadId"`
	BizID    int    `url:"biz_id"`
	Output   string `url:"output"`
	Profile  string `url:"profile"`
}
type parts_info struct {
	Partnumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}
type parts_json struct {
	Parts []parts_info `json:"parts"`
}

func upos(file *os.File, total_size int, ret upos_upload_segments) (*uploadRes, error) {
	uploadUrl := "https:" + ret.Endpoint + "/" + strings.TrimPrefix(ret.UposURI, "upos://")
	client := &http.Client{}
	req, err := http.NewRequest("POST", uploadUrl+"?uploads&output=json", nil)
	req.Header.Add("X-Upos-Auth", ret.Auth)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	t := pre_upload_json{}
	_ = json.Unmarshal(body, &t)

	segments := &chunkUploader{
		upload_id:     t.UploadID,
		chunks:        int(math.Ceil(float64(total_size) / float64(ret.ChunkSize))),
		chunk_size:    ret.ChunkSize,
		total_size:    total_size,
		threads:       Threads,
		url:           uploadUrl,
		chunk_order:   make(chan int, 5200),
		file:          file,
		Header:        req.Header,
		Maxthreads:    make(chan struct{}, Threads),
		waitGoroutine: sync.WaitGroup{},
	}

	segments.upload()
	part := parts_json{}
	for i := 0; i < segments.chunks; i++ {
		index := <-segments.chunk_order
		part.Parts = append(part.Parts, parts_info{
			Partnumber: index,
			ETag:       "etag",
		})

	}
	jsonPart, _ := json.Marshal(part)
	fmt.Println(string(jsonPart))
	params := &upload_param{
		Name:     filepath.Base(file.Name()),
		UploadId: t.UploadID,
		BizID:    ret.BizID,
		Output:   "json",
		Profile:  "ugcupos/bup",
	}
	p, _ := query.Values(params)
	for i := 0; i <= 5; i++ {
		req, _ := http.NewRequest("POST", uploadUrl, bytes.NewBuffer(jsonPart))
		req.URL.RawQuery = p.Encode()
		client := &http.Client{}
		req.Header.Add("X-Upos-Auth", ret.Auth)
		res, err := client.Do(req)
		if err != nil {
			log.Println(err, file.Name(), "第", i, "次上传失败，正在重试")
			if i == 5 {
				log.Println(err, file.Name(), "第5次上传失败")
				return nil, err
			}
			continue
		}
		body, _ := ioutil.ReadAll(res.Body)
		t := struct {
			Ok int `json:"OK"`
		}{}
		_ = json.Unmarshal(body, &t)
		if t.Ok == 1 {
			_, uposFile := filepath.Split(ret.UposURI)
			upRes := &uploadRes{
				Title:    strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name())),
				Filename: strings.TrimSuffix(filepath.Base(uposFile), filepath.Ext(uposFile)),
				Desc:     "",
			}
			return upRes, nil
		} else {
			fmt.Println(string(body))
			fmt.Println(file.Name(), "第", i, "次上传失败，正在重试")
			if i == 5 {
				fmt.Println(file.Name(), "第5次上传失败")
				return nil, errors.New("分片上传失败")
			}
		}
	}

	return nil, errors.New("分片上传失败")
}
