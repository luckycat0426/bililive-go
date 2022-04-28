package main

import (
	"context"
	"fmt"
	"github.com/bluele/gcache"
	_ "github.com/luckycat0426/bililive-go/src/cmd/bililive/internal"
	"github.com/luckycat0426/bililive-go/src/cmd/bililive/internal/flag"
	"github.com/luckycat0426/bililive-go/src/configs"
	"github.com/luckycat0426/bililive-go/src/consts"
	"github.com/luckycat0426/bililive-go/src/instance"
	"github.com/luckycat0426/bililive-go/src/listeners"
	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/log"
	"github.com/luckycat0426/bililive-go/src/pkg/biliUpload"
	"github.com/luckycat0426/bililive-go/src/pkg/events"
	"github.com/luckycat0426/bililive-go/src/pkg/utils"
	"github.com/luckycat0426/bililive-go/src/recorders"
	"github.com/luckycat0426/bililive-go/src/rpcServices"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func init() {
	if !utils.IsFFmpegExist() {
		fmt.Fprintf(os.Stderr, "FFmpeg binary not found, Please Check.\n")
		os.Exit(1)
	}
}

func getConfig() (*configs.Config, error) {
	var config *configs.Config
	config = flag.GenConfigFromFlags()
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
	inst.Mutex = new(sync.RWMutex)
	ctx := context.WithValue(context.Background(), instance.Key, inst)

	logger := log.New(ctx)
	logger.Infof("%s Version: %s Link Start", consts.AppName, consts.AppVersion)
	logger.Debugf("%+v", consts.AppInfo)
	logger.Debugf("%+v", inst.Config)

	events.NewDispatcher(ctx)

	inst.Lives = make(map[live.ID]live.Live)
	inst.Biliup = make(map[live.ID]biliUpload.Biliup)
	if err := rpcServices.NewRpcServer(ctx).Start(ctx); err != nil {
		logger.WithError(err).Fatalf("failed to init rpc server")
	}
	lm := listeners.NewManager(ctx)
	rm := recorders.NewManager(ctx)
	if err := lm.Start(ctx); err != nil {
		logger.Fatalf("failed to init listener manager, error: %s", err)
	}
	if err := rm.Start(ctx); err != nil {
		logger.Fatalf("failed to init recorder manager, error: %s", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		inst.Server.Close(ctx)
		inst.ListenerManager.Close(ctx)
		inst.RecorderManager.Close(ctx)
	}()

	inst.WaitGroup.Wait()
	logger.Info("Bye~")
}
