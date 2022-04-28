package internal

import (
	"fmt"
	"net/url"
	"time"

	"github.com/luckycat0426/bililive-go/src/live"
	"github.com/luckycat0426/bililive-go/src/pkg/utils"
)

type BaseLive struct {
	Url           *url.URL
	LastStartTime time.Time
	LiveId        live.ID
	WithUpload    bool
	UploadInfo    bool
	UploadPath    string
	Options       *live.Options
}

func genLiveId(url *url.URL) live.ID {
	return live.ID(utils.GetMd5String([]byte(fmt.Sprintf("%s%s", url.Host, url.Path))))
}

func NewBaseLive(url *url.URL, opt ...live.Option) BaseLive {
	return BaseLive{
		Url:     url,
		LiveId:  genLiveId(url),
		Options: live.MustNewOptions(opt...),
	}
}

func (a *BaseLive) GetLiveId() live.ID {
	return a.LiveId
}
func (a *BaseLive) SetUploadInfo(uploadInfo bool) {
	a.UploadInfo = uploadInfo
}
func (a *BaseLive) NeedUpload() bool {
	return a.WithUpload
}
func (a *BaseLive) GetUploadInfo() bool {
	return a.UploadInfo
}

func (a *BaseLive) SetUpload(b bool) {
	a.WithUpload = b
}
func (a *BaseLive) SetUploadPath(path string) {
	a.UploadPath = path
}
func (a *BaseLive) GetUploadPath() string {
	return a.UploadPath
}
func (a *BaseLive) GetRawUrl() string {
	return a.Url.String()
}

func (a *BaseLive) GetLastStartTime() time.Time {
	return a.LastStartTime
}

func (a *BaseLive) SetLastStartTime(time time.Time) {
	a.LastStartTime = time
}
