package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/imattdu/orbit/logx"
	"log/slog"
	"time"
)

func Marshal(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

func main() {
	//cli, _ := httpclient.New(httpclient.WithBaseURL("http://www.baidu.com"))
	//rsp := make([]byte, 0, 1)
	//_, err := cli.GetJSON(context.Background(), "abc", &rsp)
	//if err != nil {
	//	fmt.Println(err.Error())
	//}
	//fmt.Println(string(rsp))
	//ctx := context.Background()
	//ctx, s := tracex.StartSpan(ctx, "m1")
	//fmt.Println(Marshal(s))
	//ctx, s = tracex.StartSpan(ctx, "m2")
	//fmt.Println(Marshal(s))

	//err := errorx.NewBiz(errorx.ErrNotFound,
	//	errorx.WithSuccess(true),
	//	errorx.WithService(errorx.ServiceBaidu))
	//fmt.Println(errorx.ServiceOf(err))
	//fmt.Println(errorx.IsSuccess(err))
	err := logx.Init(logx.Config{
		AppName:        "matt",
		Level:          slog.LevelInfo,
		LogDir:         "./logs",
		Rotate:         logx.RotateHourly,
		MaxBackups:     10,
		QueueSize:      1024,
		ConsoleEnabled: true,
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	logx.Warn(context.Background(), "abc", map[string]interface{}{
		"a1": "def",
		"b1": "def",
	})
	time.Sleep(1 * time.Second)
}
