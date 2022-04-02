package servers

import (
	"context"
	"fmt"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"

	"github.com/luckycat0426/bililive-go/src/consts"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/recorders"
)

// FIXME: remove this
func parseInfo(ctx context.Context, l live.Live) *live.Info {
	inst := instance.GetInstance(ctx)
	obj, _ := inst.Cache.Get(l)
	info := obj.(*live.Info)
	info.Listening = inst.ListenerManager.(listeners.Manager).HasListener(ctx, l.GetLiveId())
	info.Recoding = inst.RecorderManager.(recorders.Manager).HasRecorder(ctx, l.GetLiveId())
	return info
}

func getAllLives(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	lives := liveSlice(make([]*live.Info, 0, 4))
	for _, v := range inst.Lives {
		lives = append(lives, parseInfo(r.Context(), v))
	}
	sort.Sort(lives)
	writeJSON(writer, lives)
}

func getLive(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	vars := mux.Vars(r)
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		writeMsg(writer, http.StatusNotFound, fmt.Sprintf("live id: %s can not find", vars["id"]))
		return
	}
	writeJSON(writer, parseInfo(r.Context(), live))
}

func parseLiveAction(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	vars := mux.Vars(r)
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		writeMsg(writer, http.StatusNotFound, fmt.Sprintf("live id: %s can not find", vars["id"]))
		return
	}
	switch vars["action"] {
	case "start":
		if err := inst.ListenerManager.(listeners.Manager).AddListener(r.Context(), live); err != nil {
			writeMsg(writer, http.StatusBadRequest, err.Error())
			return
		}
	case "stop":
		if err := inst.ListenerManager.(listeners.Manager).RemoveListener(r.Context(), live.GetLiveId()); err != nil {
			writeMsg(writer, http.StatusBadRequest, err.Error())
			return
		}
	default:
		writeMsg(writer, http.StatusBadRequest, fmt.Sprintf("invalid Action: %s", vars["action"]))
		return
	}
	writeJSON(writer, parseInfo(r.Context(), live))
}

/* Post data example
[
	{
		"url": "http://live.bilibili.com/1030",
		"listen": true
	},
	{
		"url": "https://live.bilibili.com/493",
		"listen": true
	}
]
*/
func addLives(writer http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	info := liveSlice(make([]*live.Info, 0))
	gjson.ParseBytes(b).ForEach(func(key, value gjson.Result) bool {
		isListen := value.Get("listen").Bool()
		u, _ := url.Parse(value.Get("url").String())
		if live, err := live.New(u, instance.GetInstance(r.Context()).Cache); err == nil {
			inst := instance.GetInstance(r.Context())
			if _, ok := inst.Lives[live.GetLiveId()]; !ok {
				inst.Lives[live.GetLiveId()] = live
				if isListen {
					inst.ListenerManager.(listeners.Manager).AddListener(r.Context(), live)
				}
				info = append(info, parseInfo(r.Context(), live))
			}
		}
		return true
	})
	sort.Sort(info)
	writeJSON(writer, info)
}

/* Post data example
[
	{
		"url": "http://live.bilibili.com/1030",
		"listen": true,
		"user":{
			"SESSDATA": "SESSDATA",
			"bili_jct": "bili_jct",
			"DedeUserID__ckMd5": "DedeUserID__ckMd5",
			"DedeUserID": "DedeUserID",
			"access_token":"access_token"
		},
        "video_info":{
            "tid": "120",
            "title":"测试视频",
            "tag":[
                "测试",
                "视频"
            ],
            "source":"youtube",
            "description":"测试视频",
            "copyright":2
        }
	},
	{
		"url": "https://live.bilibili.com/493",
		"listen": true
	}
]
*/
func addUpload(writer http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	info := liveSlice(make([]*live.Info, 0))
	gjson.ParseBytes(b).ForEach(func(key, value gjson.Result) bool {
		isListen := value.Get("listen").Bool()
		u, _ := url.Parse(value.Get("url").String())
		userInfo := value.Get("user").Map()
		user := &biliUpload.User{
			SESSDATA:        userInfo["SESSDATA"].String(),
			BiliJct:         userInfo["bili_jct"].String(),
			DedeuseridCkmd5: userInfo["DedeUserID__ckMd5"].String(),
			DedeUserID:      userInfo["DedeUserID"].String(),
			AccessToken:     userInfo["access_token"].String(),
		}
		videoInfo := value.Get("video_info").Map()
		tag := make([]string, 0)
		for _, v := range videoInfo["tag"].Array() {
			tag = append(tag, v.String())
		}
		video := &biliUpload.VideoInfos{
			Tid:         int(videoInfo["tid"].Int()),
			Title:       videoInfo["title"].String(),
			Tag:         tag,
			Source:      videoInfo["source"].String(),
			Description: videoInfo["description"].String(),
			Copyright:   int(videoInfo["copyright"].Int()),
		}
		biliup := biliUpload.Biliup{
			User:       *user,
			VideoInfos: *video,
			Threads:    4,
			Lives:      value.Get("url").String(),
		}
		if live, err := live.New(u, instance.GetInstance(r.Context()).Cache); err == nil {
			inst := instance.GetInstance(r.Context())
			if _, ok := inst.Lives[live.GetLiveId()]; !ok {
				inst.Lives[live.GetLiveId()] = live
				inst.Biliup[live.GetLiveId()] = biliup
				if isListen {
					inst.Lives[live.GetLiveId()].SetUpload(true)
					inst.ListenerManager.(listeners.Manager).AddListener(r.Context(), live)
				}
				info = append(info, parseInfo(r.Context(), live))
			} else {
				if inst.Lives[live.GetLiveId()].NeedUpload() {
					if isListen {
						inst.Lives[live.GetLiveId()].SetUpload(true)
					}
				}
			}
		}

		return true
	})
	sort.Sort(info)
	writeJSON(writer, info)

}
func getConfig(writer http.ResponseWriter, r *http.Request) {
	writeJSON(writer, instance.GetInstance(r.Context()).Config)
}

func putConfig(writer http.ResponseWriter, r *http.Request) {
	configRoom := instance.GetInstance(r.Context()).Config.LiveRooms
	configRoom = make([]string, 0, 4)
	for _, live := range instance.GetInstance(r.Context()).Lives {
		configRoom = append(configRoom, live.GetRawUrl())
	}
	instance.GetInstance(r.Context()).Config.LiveRooms = configRoom
	if err := instance.GetInstance(r.Context()).Config.Marshal(); err != nil {
		writeMsg(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeMsg(writer, http.StatusOK, "OK")
}

func removeLive(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	vars := mux.Vars(r)
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		writeMsg(writer, http.StatusNotFound, fmt.Sprintf("live id: %s can not find", vars["id"]))
		return
	}
	lm := inst.ListenerManager.(listeners.Manager)
	if lm.HasListener(r.Context(), live.GetLiveId()) {
		if err := lm.RemoveListener(r.Context(), live.GetLiveId()); err != nil {
			writeMsg(writer, http.StatusBadRequest, err.Error())
			return
		}
	}
	delete(inst.Lives, live.GetLiveId())
	writeMsg(writer, http.StatusOK, "OK")
}
func startUploadLive(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	vars := mux.Vars(r)
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		writeMsg(writer, http.StatusNotFound, fmt.Sprintf("live id: %s can not find", vars["id"]))
		return
	}
	ed := inst.EventDispatcher.(events.Dispatcher)
	ed.DispatchEvent(events.NewEvent(listeners.StartUpload, live))
	writeMsg(writer, http.StatusOK, string(live.GetLiveId()+"upload is start"))

}

func getInfo(writer http.ResponseWriter, r *http.Request) {
	writeJSON(writer, consts.AppInfo)
}
