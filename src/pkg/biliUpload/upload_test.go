package biliUpload

import (
	"fmt"
	"testing"
	"time"
)

func TestMainUpload(t *testing.T) {

	type args struct {
		uploadPath string
		Biliup     Biliup
	}
	tests := struct {
		name    string
		args    args
		wantErr bool
	}{
		name: "TestMainUpload",
		args: args{
			uploadPath: "C:\\Users\\426\\GolandProjects\\编译结果\\斗鱼\\火星东某人",
			Biliup: Biliup{
				User: User{
					SESSDATA:        "7e4269eb%2C1663017131%2Cafa75e31",
					BiliJct:         "a531b2ec5d4028df8df4bc063342f993",
					DedeuseridCkmd5: "fabd1d9358f79d0c",
					DedeUserID:      "14432590",
					AccessToken:     "9a8ddd4a7c8f569b22f4a697259b4531",
				},
				Lives:       "test.com",
				UploadLines: "ws",
				VideoInfos: VideoInfos{
					Tid:   171,
					Title: "test",
					//Tag:         []string{"test"},
					//Source:      "test",
					Copyright:   2,
					Description: "test",
				},
			},
		},
		// TODO: Add test cases.
	}

	t.Run(tests.name, func(t *testing.T) {
		if _, err := UploadFolderWithSubmit(tests.args.uploadPath, tests.args.Biliup); (err != nil) != tests.wantErr {
			t.Errorf("MainUpload() error = %v, wantErr %v", err, tests.wantErr)
		}
	})

}
func TestLock(t *testing.T) {
	b := 0
	tick := time.Tick(time.Second)
	go func() {
		for range tick {
			fmt.Println(b)
		}
	}()
	tick2 := time.Tick(time.Second * 10)
	go func() {
		i := 0
		for range tick2 {
			i++
			b = i
		}
	}()
	fmt.Println("test")
	for {
		time.Sleep(time.Second)
	}

}
