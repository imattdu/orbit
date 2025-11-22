package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/imattdu/orbit/httpclient"
	"github.com/imattdu/orbit/tracex"
)

func Marshal(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

type a func()

func t1(i int) (string, a) {
	return "", func() {
		fmt.Println(i)
	}
}

func main() {
	ctx, _ := tracex.StartSpan(context.Background(), "")
	cli, _ := httpclient.New(
		httpclient.WithBaseURL("http://www.baidu.com"),
		httpclient.WithRetry(1, nil, nil))
	rsp := make([]byte, 0, 1)
	_, err := cli.GetJSON(ctx, "abc?a=1&b=2", &rsp, httpclient.WithTimeout(time.Millisecond))
	if err != nil {
		fmt.Println(err.Error())
	}

	time.Sleep(time.Second)
}
