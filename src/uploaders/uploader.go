package uploaders

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Biliup struct {
	user         User
	lives        string
	upload_lines string
	threads      int
	Video_infos
}

func FolderUpload(folder string, u User) ([]*uploadRes, error) {
	dir, err := ioutil.ReadDir(folder)
	if err != nil {
		fmt.Printf("read dir error:%s", err)
		return nil, err
	}
	var sumbmitFiles []*uploadRes
	for _, file := range dir {
		filename := filepath.Join(folder, file.Name())
		uploadFile, err := os.Open(filename)
		if err != nil {
			fmt.Printf("open file error:%s", err)
			return nil, err
		}
		videoPart, err := upload(uploadFile, u)
		if err != nil {
			fmt.Printf("upload file error:%s", err)
			return nil, err
		}
		sumbmitFiles = append(sumbmitFiles, videoPart)
	}
	return sumbmitFiles, nil
}
