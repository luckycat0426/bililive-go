package bilibili

import (
	"errors"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	upload "github.com/luckycat0426/bililive-go/src/pkg/uploader"
	"os"
	"path/filepath"
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

func (u *Uploader) Submit(files []upload.UploadedFile) error {
	submitFiles := make([]*biliUpload.UploadRes, 0, len(files))
	for _, file := range files {
		submitFiles = append(submitFiles, file.Info.(*biliUpload.UploadRes))
	}
	err := biliUpload.Submit(*u.biliup, submitFiles)
	if err != nil {
		return err
	}
	return nil

}
func (u *Uploader) Upload(files upload.FilesNeedUpload) ([]upload.UploadedFile, error) {
	var UploadedErrors []error
	var UploadedFiles []upload.UploadedFile
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			UploadedErrors = append(UploadedErrors, err)
			continue
		}
		UploadRes, err := biliUpload.UploadFile(f, u.biliup.User, u.biliup.UploadLines)
		if err != nil {
			UploadedErrors = append(UploadedErrors, err)
			continue
		}
		UploadedFiles = append(UploadedFiles, upload.UploadedFile{
			Name: filepath.Base(file),
			Path: filepath.Dir(file),
			Info: UploadRes,
		})

	}
	var RetErr error
	if UploadedErrors != nil {
		for _, err := range UploadedErrors {
			RetErr = errors.New(err.Error() + RetErr.Error())
		}
		return UploadedFiles, RetErr
	}
	return UploadedFiles, nil
}

func (u *Uploader) Stop() error {

	return nil
}
