package biliUpload

import (
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"github.com/valyala/fasthttp"
	"net/http"
	"os"
	"sync"
)

const retry_times = 10

// var Header = http.Header{
// 	"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/63.0.3239.108"},
// 	"Referer":    []string{"https://www.bilibili.com"},
// 	"Connection": []string{"keep-alive"},
// }

type chunkUploader struct {
	upload_id     string
	chunks        int
	chunk_size    int
	total_size    int
	threads       int
	url           string
	chunk_order   chan int
	Header        http.Header
	file          *os.File
	Maxthreads    chan struct{}
	waitGoroutine sync.WaitGroup
}
type chunk_params struct {
	UploadId   string `url:"uploadId"`
	Chunks     int    `url:"chunks"`
	Total      int    `url:"total"`
	Chunk      int    `url:"chunk"`
	Size       int    `url:"size"`
	PartNumber int    `url:"partNumber"`
	Start      int    `url:"start"`
	End        int    `url:"end"`
}

func (u *chunkUploader) upload() error {

	for i := 0; i < u.chunks; i++ {
		u.Maxthreads <- struct{}{}
		buf := make([]byte, u.chunk_size)
		bufsize, _ := u.file.Read(buf)
		chunk := chunk_params{
			UploadId:   u.upload_id,
			Chunks:     u.chunks,
			Chunk:      i,
			Total:      u.total_size,
			Size:       bufsize,
			PartNumber: i + 1,
			Start:      i * u.chunk_size,
			End:        i*u.chunk_size + bufsize,
		}
		u.waitGoroutine.Add(1)
		u.uploadChunk(buf, chunk)

	}
	u.waitGoroutine.Wait()
	return nil
}
func (u *chunkUploader) uploadChunk(data []byte, params chunk_params) error {
	defer func() {
		u.waitGoroutine.Done()
		<-u.Maxthreads
	}()
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("PUT")
	req.Header.Set("X-Upos-Auth", u.Header.Get("X-Upos-Auth"))
	req.SetBodyRaw(data)
	vals, _ := query.Values(params)
	req.SetRequestURI(u.url + "?" + vals.Encode())
	for i := 0; i <= retry_times; i++ {
		err := fasthttp.Do(req, nil)
		fasthttp.ReleaseRequest(req)
		if err != nil {
			fmt.Println("上传分块出现问题，尝试重连")
			fmt.Println(err)
		} else {
			u.chunk_order <- params.PartNumber
			break
		}
		if i == retry_times {
			fmt.Println("上传分块出现问题，重试次数超过限制")
			return errors.New(string(u.chunks) + "分块上传失败")
		}
	}

	return nil
}
