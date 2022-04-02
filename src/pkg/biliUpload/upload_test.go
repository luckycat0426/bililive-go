package biliUpload

import "testing"

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
			uploadPath: "C:\\Users\\426\\GolandProjects\\bililive-go\\斗鱼\\pigff",
			Biliup: Biliup{
				User: User{
					SESSDATA:        "7e4269eb%2C1663017131%2Cafa75e31",
					BiliJct:         "a531b2ec5d4028df8df4bc063342f993",
					DedeuseridCkmd5: "fabd1d9358f79d0c",
					DedeUserID:      "14432590",
					AccessToken:     "9a8ddd4a7c8f569b22f4a697259b4531",
				},
				Lives: "test.com",
				VideoInfos: VideoInfos{
					Tid:         171,
					Title:       "test",
					Tag:         []string{"test"},
					Source:      "test",
					Copyright:   2,
					Description: "test",
				},
			},
		},
		// TODO: Add test cases.
	}

	t.Run(tests.name, func(t *testing.T) {
		if err := MainUpload(tests.args.uploadPath, tests.args.Biliup); (err != nil) != tests.wantErr {
			t.Errorf("MainUpload() error = %v, wantErr %v", err, tests.wantErr)
		}
	})

}
