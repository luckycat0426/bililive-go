package main

import (
	"context"
	"fmt"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluele/gcache"

	_ "github.com/luckycat0426/bililive-go/src/cmd/bililive/internal"
	"github.com/luckycat0426/bililive-go/src/cmd/bililive/internal/flag"
	"github.com/luckycat0426/bililive-go/src/configs"
	"github.com/luckycat0426/bililive-go/src/consts"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/log"
	"github.com/luckycat0426/bililive-go/src/metrics"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
	"github.com/luckycat0426/bililive-go/src/pkg/utils"
	"github.com/luckycat0426/bililive-go/src/recorders"
	"github.com/luckycat0426/bililive-go/src/servers"
)

func init() {
	if !utils.IsFFmpegExist() {
		fmt.Fprintf(os.Stderr, "FFmpeg binary not found, Please Check.\n")
		os.Exit(1)
	}
}

func getConfig() (*configs.Config, error) {
	var config *configs.Config
	if *flag.Conf != "" {
		c, err := configs.NewConfigWithFile(*flag.Conf)
		if err != nil {
			return nil, err
		}
		config = c
	} else {
		config = flag.GenConfigFromFlags()
	}
	return config, config.Verify()
}

func main() {
	config, err := getConfig()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	inst := new(instance.Instance)
	inst.Config = config
	inst.Cache = gcache.New(128).LRU().Build()
	ctx := context.WithValue(context.Background(), instance.Key, inst)

	logger := log.New(ctx)
	logger.Infof("%s Version: %s Link Start", consts.AppName, consts.AppVersion)
	logger.Debugf("%+v", consts.AppInfo)
	logger.Debugf("%+v", inst.Config)

	events.NewDispatcher(ctx)

	inst.Lives = make(map[live.ID]live.Live)
	inst.Biliup = make(map[live.ID]biliUpload.Biliup)
	for _, room := range inst.Config.LiveRooms {
		u, err := url.Parse(room)
		if err != nil {
			logger.WithField("url", room).Error(err)
			continue
		}
		opts := make([]live.Option, 0)
		if v, ok := inst.Config.Cookies[u.Host]; ok {
			opts = append(opts, live.WithKVStringCookies(u, v))
		}
		l, err := live.New(u, inst.Cache, opts...)
		if err != nil {
			logger.WithField("url", room).Error(err.Error())
			continue
		}
		if _, ok := inst.Lives[l.GetLiveId()]; ok {
			logger.Errorf("%s is exist!", room)
			continue
		}
		inst.Lives[l.GetLiveId()] = l
	}

	if inst.Config.RPC.Enable {
		if err := servers.NewServer(ctx).Start(ctx); err != nil {
			logger.WithError(err).Fatalf("failed to init server")
		}
	}
	lm := listeners.NewManager(ctx)
	rm := recorders.NewManager(ctx)
	if err := lm.Start(ctx); err != nil {
		logger.Fatalf("failed to init listener manager, error: %s", err)
	}
	if err := rm.Start(ctx); err != nil {
		logger.Fatalf("failed to init recorder manager, error: %s", err)
	}

	if err = metrics.NewCollector(ctx).Start(ctx); err != nil {
		logger.Fatalf("failed to init metrics collector, error: %s", err)
	}

	for _, _live := range inst.Lives {
		if err := lm.AddListener(ctx, _live); err != nil {
			logger.WithFields(map[string]interface{}{"url": _live.GetRawUrl()}).Error(err)
		}
		time.Sleep(time.Second * 5)
	}

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		if inst.Config.RPC.Enable {
			inst.Server.Close(ctx)
		}
		inst.ListenerManager.Close(ctx)
		inst.RecorderManager.Close(ctx)
	}()

	inst.WaitGroup.Wait()
	logger.Info("Bye~")
}
