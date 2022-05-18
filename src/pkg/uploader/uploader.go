package upload

import (
	"errors"
)

type FilesNeedUpload []string

type UploadedFile struct {
	Name string
	Size int64
	Path string
	Type string
	Info interface{}
}
type ID string
type Upload interface {
	Upload(FilesNeedUpload) ([]UploadedFile, error)
	Submit([]UploadedFile) error
	Stop() error
}

type Builder interface {
	Build(interface{}) (Upload, error)
}

var m = make(map[string]Builder)

func Register(name string, b Builder) {
	m[name] = b
}

func New(name string, uploader interface{}) (Upload, error) {
	builder, ok := m[name]
	if !ok {
		return nil, errors.New("unknown Uploader")
	}
	return builder.Build(uploader)
}
