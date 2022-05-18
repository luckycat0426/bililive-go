package uploaders

import "github.com/luckycat0426/bililive-go/src/pkg/events"

const (
	UploaderStart   events.EventType = "UploaderStart"
	UploaderStop    events.EventType = "UploaderStop"
	UploadEnd       events.EventType = "UploadEnd" //每个视频段上传结束触发事件
	UploaderRestart events.EventType = "UploaderRestart"
)
