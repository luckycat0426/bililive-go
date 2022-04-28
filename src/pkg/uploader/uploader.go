package upload

import (
	"errors"
	"github.com/luckycat0426/bililive-go/src/live"
)

type FilesNeedUpload []string

type UploadedFile struct {
	Name string
	Size int64
	Path string
	Type string
	info interface{}
}
type ID string
type Upload interface {
	Upload(FilesNeedUpload, live.Live) ([]UploadedFile, error)
	Stop() error
}

type Builder interface {
	Build(interface{}) (Upload, error)
}

var m = make(map[string]Builder)

func Register(name string, b Builder) {
	m[name] = b
}

func New(name string, cfg map[string]string) (Upload, error) {
	builder, ok := m[name]
	if !ok {
		return nil, errors.New("unknown Uploader")
	}
	return builder.Build(cfg)
}
