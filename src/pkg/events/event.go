package events

import (
	"github.com/luckycat0426/bililive-go/src/live"
	upload "github.com/luckycat0426/bililive-go/src/pkg/uploader"
)

type EventType string

type EventHandler func(event *Event)
type UploadObject struct {
	Live       live.Live
	UploadInfo interface{}
}
type UploadEndObject struct {
	Live          live.Live
	Uploader      upload.Upload
	UploadedFiles []upload.UploadedFile
}
type Event struct {
	Type   EventType
	Object interface{}
}

func NewEvent(eventType EventType, object interface{}) *Event {
	return &Event{eventType, object}
}

type EventListener struct {
	Handler EventHandler
}

func NewEventListener(handler EventHandler) *EventListener {
	return &EventListener{handler}
}
