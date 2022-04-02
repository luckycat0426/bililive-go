package instance

import (
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"sync"

	"github.com/bluele/gcache"

	"github.com/luckycat0426/bililive-go/src/configs"
	"github.com/luckycat0426/bililive-go/src/interfaces"
	"github.com/luckycat0426/bililive-go/src/live"
)

type Instance struct {
	WaitGroup       sync.WaitGroup
	Config          *configs.Config
	Logger          *interfaces.Logger
	Lives           map[live.ID]live.Live
	Biliup          map[live.ID]biliUpload.Biliup
	Cache           gcache.Cache
	Server          interfaces.Module
	EventDispatcher interfaces.Module
	ListenerManager interfaces.Module
	RecorderManager interfaces.Module
	UploaderManager interfaces.Module
}
