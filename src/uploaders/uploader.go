//go:generate mockgen -package recorders -destination mock_test.go github.com/luckycat0426/bililive-go/src/recorders Recorder,Manager
package uploaders

import (
	"context"
	"fmt"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	upload "github.com/luckycat0426/bililive-go/src/pkg/uploader"
	"github.com/luckycat0426/bililive-go/src/pkg/uploader/bilibili"
	"github.com/luckycat0426/bililive-go/src/recorders"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"

	"github.com/luckycat0426/bililive-go/src/configs"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/interfaces"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
	"github.com/luckycat0426/bililive-go/src/pkg/utils"
)

const (
	begin uint32 = iota
	pending
	running
	stopped
)

// for test
var (
	newUploadWebSite = func(Uploader string, info interface{}) (upload.Upload, error) {
		UploaderName := bilibili.Name
		UploaderInfo := info.(*biliUpload.Biliup)
		if Uploader != "" {
			UploaderName = Uploader
		}
		return upload.New(UploaderName, UploaderInfo)
	}

	mkdir = func(path string) error {
		return os.MkdirAll(path, os.ModePerm)
	}

	removeEmptyFile = func(file string) {
		if stat, err := os.Stat(file); err == nil && stat.Size() == 0 {
			os.Remove(file)
		}
	}
	removeUploadedFile = func(files []upload.UploadedFile) {
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(file.Path, file.Name)); err == nil {
				os.Remove(filepath.Join(file.Path, file.Name))
			}

		}
	}
)

var defaultFileNameTmpl = template.Must(template.New("filename").Funcs(utils.GetFuncMap()).
	Parse(`{{ .Live.GetPlatformCNName }}/{{ .HostName | filenameFilter }}/[{{ now | date "2006-01-02 15-04-05"}}][{{ .HostName | filenameFilter }}][{{ .RoomName | filenameFilter }}].flv`))

type Uploader interface {
	Start() error
	StartTime() time.Time
	Close()
}

type uploader struct {
	Uploader       upload.Upload
	UploadedFiles  []upload.UploadedFile
	UploadFilesDir string
	recorder       recorders.Recorder
	config         *configs.Config
	ed             events.Dispatcher
	Live           live.Live
	logger         *interfaces.Logger
	cache          gcache.Cache
	startTime      time.Time
	uploadRLock    *sync.RWMutex
	stop           chan struct{}
	state          uint32
}

func NewUploader(ctx context.Context, u upload.Upload, live live.Live) (Uploader, error) {
	inst := instance.GetInstance(ctx)
	r, err := inst.RecorderManager.(recorders.Manager).GetRecorder(ctx, live.GetLiveId())
	if err != nil {
		return nil, fmt.Errorf("live Recorder not exist,Init Uploader Error:%s", err.Error())
	}
	return &uploader{
		Uploader:       u,
		UploadFilesDir: filepath.Dir(r.GetRecordingFileName()),
		UploadedFiles:  []upload.UploadedFile{},
		recorder:       r,
		config:         inst.Config,
		Live:           live,
		cache:          inst.Cache,
		startTime:      time.Now(),
		ed:             inst.EventDispatcher.(events.Dispatcher),
		logger:         inst.Logger,
		state:          begin,
		stop:           make(chan struct{}),
		uploadRLock:    new(sync.RWMutex),
	}, nil
}

func (u *uploader) tryUpload() {

	files, err := ioutil.ReadDir(u.UploadFilesDir)
	if err != nil {
		u.logger.Errorf("read dir error:%s", err.Error())
		return
	}
	uploadFile := make(upload.FilesNeedUpload, 0, len(files))
	u.uploadRLock.RLock()
	r := u.GetRecorder()
	RecordingFile := ""
	if r != nil {
		RecordingFile = r.GetRecordingFileName()
	}
	u.uploadRLock.RUnlock()
	for _, file := range files {
		if file.Name() == filepath.Base(RecordingFile) {
			u.getLogger().Debugf("file %s is recording file,skip", RecordingFile)
			continue
		}
		u.getLogger().Debugf("add file %s %d MB to upload list", file.Name(), file.Size()/1024/1024)
		if int(file.Size()) < u.config.Feature.UploadThresholdSize {
			u.getLogger().Debugf("%s is too small,ignore it,minimal uploaded size is %d,", file.Name(), u.config.Feature.UploadThresholdSize)
			continue
		}
		uploadFile = append(uploadFile, filepath.Join(u.UploadFilesDir, file.Name()))

	}
	if len(uploadFile) == 0 {
		if r == nil {
			u.getLogger().Infof("There is no Recorder for Uploader,sumbit video and remove Uploader")
			u.ed.DispatchEvent(events.NewEvent(listeners.UploadEnd, events.UploadEndObject{
				Live:          u.Live,
				Uploader:      u.Uploader,
				UploadedFiles: u.UploadedFiles,
			}))
			time.Sleep(time.Second * 60)
			return
		}
		u.getLogger().Debugf("no files to upload,sleep 60s")
		time.Sleep(time.Second * 60)
		return
	}
	u.startTime = time.Now()
	UploadedFiles, err := u.Uploader.Upload(uploadFile)
	if err != nil {
		u.getLogger().Error(err)
	}
	for _, v := range UploadedFiles {
		u.getLogger().Debugf("upload file %s success", v.Name)
		u.getLogger().Debugln(v.Info)
	}
	u.UploadedFiles = append(u.UploadedFiles, UploadedFiles...)
	removeUploadedFile(UploadedFiles)
	u.ed.DispatchEvent(events.NewEvent(listeners.UploadEnd, events.UploadEndObject{
		Live:          u.Live,
		Uploader:      u.Uploader,
		UploadedFiles: u.UploadedFiles,
	}))
	//removeEmptyFile(fileName)
}

func (u *uploader) run() {
	for {
		select {
		case <-u.stop:
			return
		default:
			u.tryUpload()
		}
	}
}

func (u *uploader) getUploader() upload.Upload {
	u.uploadRLock.RLock()
	defer u.uploadRLock.RUnlock()
	return u.Uploader
}

//func (u *uploader) setAndCloseParser(p parser.Parser) {
//	u.parserLock.Lock()
//	defer u.parserLock.Unlock()
//	if r.parser != nil {
//		r.parser.Stop()
//	}
//	r.parser = p
//}

func (u *uploader) Start() error {
	if !atomic.CompareAndSwapUint32(&u.state, begin, pending) {
		return nil
	}
	go u.run()
	u.getLogger().Info("Uploader Start")
	u.ed.DispatchEvent(events.NewEvent(UploaderStart, u.Live))
	atomic.CompareAndSwapUint32(&u.state, pending, running)
	return nil
}

func (u *uploader) StartTime() time.Time {
	return u.startTime
}

func (u *uploader) Close() {
	if !atomic.CompareAndSwapUint32(&u.state, running, stopped) {
		return
	}
	close(u.stop)
	if p := u.getUploader(); p != nil {
		p.Stop()
	}
	u.getLogger().Info("Upload End")
	u.ed.DispatchEvent(events.NewEvent(UploaderStop, u.Live))
}

func (u *uploader) getLogger() *logrus.Entry {
	return u.logger.WithFields(u.getFields())
}

func (u *uploader) getFields() map[string]interface{} {
	obj, err := u.cache.Get(u.Live)
	if err != nil {
		return nil
	}
	info := obj.(*live.Info)
	return map[string]interface{}{
		"host": info.HostName,
		"room": info.RoomName,
	}
}
func (u *uploader) GetRecorder() recorders.Recorder {
	u.uploadRLock.RLock()
	defer u.uploadRLock.RUnlock()
	return u.recorder
}
func (u *uploader) SetRecorder(r recorders.Recorder) {
	u.uploadRLock.Lock()
	defer u.uploadRLock.Unlock()
	u.recorder = r
}
