package biliUpload

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/google/go-querystring/query"
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
		go u.upload_chunk(buf, chunk)
	}
	u.waitGoroutine.Wait()
	return nil
}
func (u *chunkUploader) upload_chunk(data []byte, params chunk_params) error {
	u.Maxthreads <- struct{}{}
	defer func() {
		<-u.Maxthreads
	}()
	var msg string
	client := &http.Client{}
	vals, _ := query.Values(params)
	req, err := http.NewRequest("PUT", u.url, bytes.NewBuffer(data))
	req.URL.RawQuery = vals.Encode()
	req.Header = u.Header
	if err != nil {
		return err
	}
	for i := 0; i <= retry_times; i++ {
		res, err := client.Do(req)
		if err != nil {
			msg = "上传出现问题，尝试重连"
			fmt.Println(msg)
			fmt.Println(err)
			res.Body.Close()
			// return err
		} else {
			u.chunk_order <- params.PartNumber
			res.Body.Close()
			break
		}
		if i == retry_times {
			return err
		}
	}
	u.waitGoroutine.Done()
	return nil
}
