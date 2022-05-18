package uploaders

import "errors"

var (
	ErrUploaderExist            = errors.New("Uploader is exist")
	ErrUploaderNotExist         = errors.New("Uploader is not exist")
	ErrUploaderNotSupportStatus = errors.New("Uploader not support get status")
)
