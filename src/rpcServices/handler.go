package rpcServices

import (
	"encoding/json"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"net/url"
	"time"
)

type RecordService struct {
	UnimplementedRecordServiceServer
}
type Status struct {
	Code   int
	Record bool
	Upload bool
}

func (a *RecordService) Record(req *RecordRequest, stream RecordService_RecordServer) error {
	ctx := stream.Context()
	u, _ := url.Parse(req.GetRecordUrl())
	marshalJson, _ := req.GetBiliup().MarshalJSON()
	var biliup biliUpload.Biliup
	json.Unmarshal(marshalJson, &biliup)
	inst := instance.GetInstance(ctx)
	l, err := live.New(u, instance.GetInstance(ctx).Cache)
	if err != nil {
		if _, ok := inst.Lives[l.GetLiveId()]; ok {
			inst.Lives[l.GetLiveId()] = l
			inst.Biliup[l.GetLiveId()] = biliup
			inst.ListenerManager.(listeners.Manager).AddListener(ctx, l)
		}
	}

	ticker := time.Tick(1 * time.Second)
	startUpload := false
	taskEnd := false
	for range ticker {
		info, _ := l.GetInfo()
		uploadInfo := l.GetUploadInfo()
		if uploadInfo && !startUpload {
			startUpload = true
			go func() {
				biliUpload.MainUpload(l.GetUploadPath(), biliup)
				taskEnd = true
			}()
		}
		res := &RecordResponse{
			Code:         200,
			RecordStatus: info.Recoding,
			UploadStatus: uploadInfo,
		}
		if err := stream.Send(res); err != nil {
			inst.Logger.Errorf(err.Error())
			return err
		}
		if taskEnd {
			lm := inst.ListenerManager.(listeners.Manager)
			if lm.HasListener(ctx, l.GetLiveId()) {
				if err := lm.RemoveListener(ctx, l.GetLiveId()); err != nil {
					inst.Logger.Errorf(err.Error())
					return err
				}
			}
			delete(inst.Lives, l.GetLiveId())
			break
		}
	}
	return nil
}
