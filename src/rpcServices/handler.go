package rpcServices

import (
	"context"
	"encoding/json"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"github.com/luckycat0426/bililive-go/src/recorders"
	"github.com/luckycat0426/bililive-go/src/uploaders"
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

	ctx := stream.(*recordServiceRecordServer).ServerStream.(*serverStream).ctx
	u, _ := url.Parse(req.GetRecordUrl())
	marshalJson, _ := req.GetBiliup().MarshalJSON()
	var biliup biliUpload.Biliup
	json.Unmarshal(marshalJson, &biliup)
	inst := instance.GetInstance(ctx)
	inst.Logger.Info("Receiving Record Request from client,Start Recording")
	l, err := live.New(u, instance.GetInstance(ctx).Cache)
	if err != nil {
		return err
	}
	inst.Mutex.Lock()
	if _, ok := inst.Lives[l.GetLiveId()]; !ok {
		inst.Lives[l.GetLiveId()] = l
		inst.Biliup[l.GetLiveId()] = biliup
		err := inst.ListenerManager.(listeners.Manager).AddListener(ctx, l)
		defer func(manager listeners.Manager, ctx context.Context, liveId live.ID) {
			inst.Mutex.Lock()
			delete(inst.Lives, l.GetLiveId())
			inst.Mutex.Unlock()
			err := manager.RemoveListener(ctx, liveId)
			if err != nil {
				inst.Logger.Error(err)
			}
		}(inst.ListenerManager.(listeners.Manager), ctx, l.GetLiveId())
		if err != nil {
			inst.Mutex.Unlock()
			return err
		}
	}
	inst.Mutex.Unlock()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	WaitTimes := 20 //record与upload都停止后的等待时间,为ticker的倍数
	Times := 0
	LiveId := l.GetLiveId()
	for range ticker.C {
		recordInfo := inst.RecorderManager.(recorders.Manager).HasRecorder(ctx, LiveId)
		uploadInfo := inst.UploaderManager.(uploaders.Manager).HasUploader(ctx, LiveId)
		res := &RecordResponse{
			Code:         200,
			RecordStatus: recordInfo,
			UploadStatus: uploadInfo,
			Msg:          "record and upload status",
		}
		if !recordInfo && !uploadInfo {
			Times++
		} else {
			Times = 0
		}
		if err := stream.Send(res); err != nil {
			inst.Logger.Errorln(err.Error())
			return err
		}
		if Times > WaitTimes {

			if err := stream.Send(&RecordResponse{
				Code: 200,
				Msg:  "Finish",
			}); err != nil {
				inst.Logger.Errorln(err.Error())
				return err
			}
			//stream.Context().Done()
			break
		}
	}
	return nil
}
