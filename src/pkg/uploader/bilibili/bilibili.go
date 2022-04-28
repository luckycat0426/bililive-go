package bilibili

import (
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	upload "github.com/luckycat0426/bililive-go/src/pkg/uploader"
	"os"
)

const (
	Name = "bilibili"
)

func init() {
	upload.Register(Name, new(builder))
}

type builder struct{}

func (b *builder) Build(UploadInfo interface{}) (upload.Upload, error) {
	return &Uploader{
		biliup: UploadInfo.(*biliUpload.Biliup),
	}, nil
}

type Uploader struct {
	biliup *biliUpload.Biliup
}

func (u *Uploader) Upload(files upload.FilesNeedUpload, live live.Live) ([]upload.UploadedFile, error) {
	var UploadedErrors []error
	var UploadedFiles []upload.UploadedFile
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			UploadedErrors = append(UploadedErrors, err)
			continue
		}
		UploadRes,_ := biliUpload.UploadFile(f, u.biliup.User, u.biliup.UploadLines)
		if err!=nil{
			UploadedErrors = append(UploadedErrors, err)
			continue
		}
		UploadedFiles = append(UploadedFiles, upload.UploadedFile{
			Name :file
			FileUrl:  UploadRes.Url,
		})

	}
	return nil, nil
}

func (u *Uploader) Stop() error {

	return nil
}
