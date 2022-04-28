package recorders

import (
	"context"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"os"
	"sync"
	"time"

	"github.com/luckycat0426/bililive-go/src/configs"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/interfaces"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
)

func NewManager(ctx context.Context) Manager {
	rm := &manager{
		savers: make(map[live.ID]Recorder),
		cfg:    instance.GetInstance(ctx).Config,
	}
	instance.GetInstance(ctx).RecorderManager = rm

	return rm
}

type Manager interface {
	interfaces.Module
	AddRecorder(ctx context.Context, live live.Live) error
	RemoveRecorder(ctx context.Context, liveId live.ID) error
	RestartRecorder(ctx context.Context, liveId live.Live) error
	GetRecorder(ctx context.Context, liveId live.ID) (Recorder, error)
	HasRecorder(ctx context.Context, liveId live.ID) bool
}

// for test
var (
	newRecorder = NewRecorder
)

type manager struct {
	lock   sync.RWMutex
	savers map[live.ID]Recorder
	cfg    *configs.Config
}

func (m *manager) registryListener(ctx context.Context, ed events.Dispatcher) {
	ed.AddEventListener(listeners.LiveStart, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if err := m.AddRecorder(ctx, live); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to add recorder, err: %v", err)
		}
	}))

	ed.AddEventListener(listeners.RoomNameChanged, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		if err := m.RestartRecorder(ctx, live); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to cronRestart recorder, err: %v", err)
		}
	}))
	ed.AddEventListener(listeners.StartUpload, events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		inst := instance.GetInstance(ctx)
		inst.Mutex.RLock()
		b, ok := inst.Biliup[live.GetLiveId()]
		inst.Mutex.RUnlock()
		if ok {
			live.SetUploadInfo(true)
			defer live.SetUploadInfo(false)
			uploadedFile, err := biliUpload.UploadFolderWithSubmit(live.GetUploadPath(), b)
			if err != nil {
				inst.Logger.Errorf("failed to upload folder, err: %v", err)
			} else {
				for _, v := range uploadedFile {
					inst.Logger.Infof("file: %s uploaded, deleting", v.FileName)
					err := os.Remove(v.FilePath + v.FileName)
					if err != nil {
						inst.Logger.Errorf("failed to delete file: %s, err: %v", v.FileName, err)
					}
				}

			}
		} else {
			inst.Logger.Errorf("failed to find UploadInfo for live: %v", live.GetLiveId())
		}
	}))
	//ed.AddEventListener(listeners.StartUploadWithDelay, events.NewEventListener(func(event *events.Event) {
	//	time.Sleep(time.Minute * 3)
	//	live := event.Object.(live.Live)
	//	inst := instance.GetInstance(ctx)
	//	if b, ok := inst.Biliup[live.GetLiveId()]; ok {
	//		live.SetUploadInfo(true)
	//		biliUpload.MainUpload(live.GetUploadPath(), b)
	//	} else {
	//		inst.Logger.Errorf("failed to find UploadInfo for live: %v", live.GetLiveId())
	//	}
	//	live.SetUploadInfo(false)
	//}))

	removeEvtListener := events.NewEventListener(func(event *events.Event) {
		live := event.Object.(live.Live)
		if !m.HasRecorder(ctx, live.GetLiveId()) {
			return
		}
		if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
			instance.GetInstance(ctx).Logger.Errorf("failed to remove recorder, err: %v", err)
		}
	})
	ed.AddEventListener(listeners.LiveEnd, removeEvtListener)
	ed.AddEventListener(listeners.ListenStop, removeEvtListener)
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
	for id, recorder := range m.savers {
		recorder.Close()
		delete(m.savers, id)
	}
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
}

func (m *manager) AddRecorder(ctx context.Context, live live.Live) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.savers[live.GetLiveId()]; ok {
		return ErrRecorderExist
	}
	recorder, err := newRecorder(ctx, live)
	if err != nil {
		return err
	}
	m.savers[live.GetLiveId()] = recorder

	if maxDur := m.cfg.VideoSplitStrategies.MaxDuration; maxDur != 0 {
		go m.cronRestart(ctx, live)
	}
	return recorder.Start()
}

func (m *manager) cronRestart(ctx context.Context, live live.Live) {
	recorder, err := m.GetRecorder(ctx, live.GetLiveId())
	if err != nil {
		return
	}
	if time.Now().Sub(recorder.StartTime()) < m.cfg.VideoSplitStrategies.MaxDuration {
		time.AfterFunc(time.Minute/4, func() {
			m.cronRestart(ctx, live)
		})
		return
	}
	if err := m.RestartRecorder(ctx, live); err != nil {
		return
	}
}

func (m *manager) RestartRecorder(ctx context.Context, live live.Live) error {
	if err := m.RemoveRecorder(ctx, live.GetLiveId()); err != nil {
		return err
	}
	if err := m.AddRecorder(ctx, live); err != nil {
		return err
	}
	return nil
}

func (m *manager) RemoveRecorder(ctx context.Context, liveId live.ID) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	recorder, ok := m.savers[liveId]
	if !ok {
		return ErrRecorderNotExist
	}
	recorder.Close()
	delete(m.savers, liveId)
	return nil
}

func (m *manager) GetRecorder(ctx context.Context, liveId live.ID) (Recorder, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r, ok := m.savers[liveId]
	if !ok {
		return nil, ErrRecorderNotExist
	}
	return r, nil
}

func (m *manager) HasRecorder(ctx context.Context, liveId live.ID) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.savers[liveId]
	return ok
}
