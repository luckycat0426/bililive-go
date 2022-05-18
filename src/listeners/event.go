package listeners

import (
	"github.com/luckycat0426/bililive-go/src/pkg/events"
)

const (
	ListenStart     events.EventType = "ListenStart"
	ListenStop      events.EventType = "ListenStop"
	LiveStart       events.EventType = "LiveStart"
	LiveEnd         events.EventType = "LiveEnd"
	StartUpload     events.EventType = "StartUpload"
	UploadEnd       events.EventType = "UploadEnd"
	UploadStart     events.EventType = "UploadStart"
	RemoveRecorder  events.EventType = "RemoveRecorder"
	RestartRecorder events.EventType = "RestartRecorder"
	//StartUploadWithDelay events.EventType = "StartUploadWithDelay"
	RoomNameChanged events.EventType = "RoomNameChanged"
)
