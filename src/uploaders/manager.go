package uploaders

import (
	"bytes"
	"context"
	"fmt"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/interfaces"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
	"github.com/luckycat0426/bililive-go/src/pkg/utils"
	"github.com/luckycat0426/bililive-go/src/recorders"
	"sync"
	"text/template"
	"time"
)

func NewManager(ctx context.Context) Manager {
	um := &manager{
		savers: make(map[live.ID]Uploader),
	}
	instance.GetInstance(ctx).UploaderManager = um
	return um
}

var defaultTitleTmpl = template.Must(template.New("filename").Funcs(utils.GetFuncMap()).
	Parse(`{{ .Live.GetPlatformCNName }}/{{ .HostName | filenameFilter }}/[{{ now | date "2006-01-02 15-04-05"}}][{{ .HostName | filenameFilter }}][{{ .RoomName | filenameFilter }}].flv`))

type Manager interface {
	interfaces.Module
	AddUploader(ctx context.Context, u interface{}, live live.Live) error
	RemoveUploader(ctx context.Context, liveId live.ID) error
	RestartUploader(ctx context.Context, u interface{}, live live.Live) error
	GetUploader(ctx context.Context, liveId live.ID) (Uploader, error)
	HasUploader(ctx context.Context, liveId live.ID) bool
}

// for test
var (
	newUploader = NewUploader
)

type manager struct {
	lock   sync.RWMutex
	savers map[live.ID]Uploader
}

func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	ed.AddEventListener(listeners.UploadStart, events.NewEventListener(func(event *events.Event) {
		l := event.Object.(live.Live)
		inst := instance.GetInstance(ctx)
		inst.Mutex.Lock()
		u := instance.GetInstance(ctx).Biliup[l.GetLiveId()]
		obj, _ := inst.Cache.Get(l)
		info := obj.(*live.Info)
		tmpl := defaultTitleTmpl
		_tmpl, err := template.New("user_filename").Funcs(utils.GetFuncMap()).Parse(u.Title)
		if err == nil {
			tmpl = _tmpl
		}
		buf := new(bytes.Buffer)
		if err = tmpl.Execute(buf, info); err != nil {
			inst.Logger.Debugf("failed to render Title, err: %v,use origin title string", err)
		} else {
			u.Title = buf.String()
		}
		inst.Mutex.Unlock()
		if err := m.AddUploader(ctx, &u, l); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to add upload, err: %v", err)
		}
	}))
	ed.AddEventListener(listeners.RestartRecorder, events.NewEventListener(func(event *events.Event) {
		l := event.Object.(live.Live)
		inst := instance.GetInstance(ctx)
		up, err := m.GetUploader(ctx, l.GetLiveId())
		if err != nil {
			inst.Logger.Errorf("failed to get uploader, err: %v", err)
			return
		}
		r, err := inst.RecorderManager.(recorders.Manager).GetRecorder(ctx, l.GetLiveId())
		if err != nil {
			inst.Logger.Errorf("failed to get recorder, err: %v", err)
			return
		}
		up.(*uploader).SetRecorder(r)
		inst.Logger.Debugf("Recorder restarted,change uplerder's recorder to new recorder")
	}))
	ed.AddEventListener(listeners.RemoveRecorder, events.NewEventListener(func(event *events.Event) {
		l := event.Object.(live.Live)
		inst := instance.GetInstance(ctx)
		up, err := m.GetUploader(ctx, l.GetLiveId())
		if err != nil {
			inst.Logger.Errorf("failed to get uploader, err: %v", err)
			return
		}
		up.(*uploader).SetRecorder(nil)
		inst.Logger.Debugf("Set Uploader's recorder to nil")
	}))
	ed.AddEventListener(listeners.LiveEnd, events.NewEventListener(func(event *events.Event) {
		MaxWaitTime := 2
		inst := instance.GetInstance(ctx)
		inst.Logger.Infof("Live end ,Wait Max %v Hour to remove Uploader", MaxWaitTime)
		time.Sleep(time.Hour * time.Duration(MaxWaitTime))
		live := event.Object.(live.Live)
		up, err := m.GetUploader(ctx, live.GetLiveId())
		if err != nil {
			inst.Logger.Debugf("Uploader is not exist, can't remove uploader and submit video")
			return
		}
		if err := up.(*uploader).Uploader.Submit(up.(*uploader).UploadedFiles); err != nil {
			inst.Logger.Errorf("failed to submit video, err: %v \n,there are Uploaded files info,try to upload manully", err)
			for _, f := range up.(*uploader).UploadedFiles {
				inst.Logger.Errorln(f)
			}
		}
		if err := m.RemoveUploader(ctx, live.GetLiveId()); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to remove upload, err: %v", err)
		}
	}))

	removeEvtListener := events.NewEventListener(func(event *events.Event) {
		ob := event.Object.(events.UploadEndObject)
		inst := instance.GetInstance(ctx)
		if inst.RecorderManager.(recorders.Manager).HasRecorder(ctx, ob.Live.GetLiveId()) {
			inst.Logger.Debugf("live %s is still recording, can't remove uploader and sumbit video", ob.Live.GetRawUrl())
			return
		}
		if !m.HasUploader(ctx, ob.Live.GetLiveId()) {
			inst.Logger.Debugf("Uploader is not exist, can't remove uploader and submit video")
			return
		}
		if err := ob.Uploader.Submit(ob.UploadedFiles); err != nil {
			inst.Logger.Errorf("failed to submit video, err: %v \n,there are Uploaded files info,try to upload manully", err)
			for _, f := range ob.UploadedFiles {
				inst.Logger.Errorln(f)
				inst.Logger.Errorln(&f.Info)
			}
		}
		inst.Logger.Infoln("Success Submit Video", ob.Live.GetRawUrl())
		if err := m.RemoveUploader(ctx, ob.Live.GetLiveId()); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to remove upload, err: %v", err)
		}

	})
	ed.AddEventListener(listeners.UploadEnd, removeEvtListener)
}

func (m *manager) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	inst.Mutex.RLock()
	if inst.Config.RPC.Enable || len(inst.Lives) > 0 {
		inst.WaitGroup.Add(1)
	}
	inst.Mutex.RUnlock()
	m.registryListener(ctx, inst.EventDispatcher.(events.Dispatcher))
	return nil
}

func (m *manager) Close(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for id, uploader := range m.savers {
		uploader.Close()
		delete(m.savers, id)
	}
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
}

func (m *manager) AddUploader(ctx context.Context, uploadInfo interface{}, live live.Live) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.savers[live.GetLiveId()]; ok {
		return ErrUploaderExist
	}
	u, err := newUploadWebSite("bilibili", uploadInfo)
	if err != nil {
		return fmt.Errorf("failed to init Uploader bilibili, err: %v", err)
	}
	uploader, err := newUploader(ctx, u, live)
	if err != nil {
		return err
	}
	m.savers[live.GetLiveId()] = uploader
	return uploader.Start()
}

func (m *manager) RestartUploader(ctx context.Context, uploadInfo interface{}, live live.Live) error {
	if err := m.RemoveUploader(ctx, live.GetLiveId()); err != nil {
		return err
	}
	if err := m.AddUploader(ctx, uploadInfo, live); err != nil {
		return err
	}
	return nil
}

func (m *manager) RemoveUploader(ctx context.Context, liveId live.ID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	uploader, ok := m.savers[liveId]
	if !ok {
		return ErrUploaderNotExist
	}
	uploader.Close()
	delete(m.savers, liveId)
	return nil
}

func (m *manager) GetUploader(ctx context.Context, liveId live.ID) (Uploader, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r, ok := m.savers[liveId]
	if !ok {
		return nil, ErrUploaderNotExist
	}
	return r, nil
}

func (m *manager) HasUploader(ctx context.Context, liveId live.ID) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.savers[liveId]
	return ok
}
