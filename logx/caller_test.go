package logx

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

func testCallerF0() {
	c := getCaller()
	fmt.Printf("%#v\n", c)
}

func testCallerF1() {
	testCallerF0()
}

func testCallerF2() {
	testCallerF1()
}

func TestGetCaller(t *testing.T) {
	// logger -> info -> log -> handler 4层
	testCallerF2()
}

// TestTrimFilePath 包名+文件
func TestTrimFilePath(t *testing.T) {
	_, file, _, _ := runtime.Caller(1)
	fmt.Printf("before:%s\n", file)
	tf := trimFilePath(file)
	fmt.Printf("after:%s\n", tf)

	_, short := filepath.Split(file)
	fmt.Printf("short:%s\n", short)
}

func testTrimFuncName() {
	pc, _, _, _ := runtime.Caller(1)
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName := trimFuncName(fn.Name())
		fmt.Println(funcName)
	}
}

// TestTrimFuncName
func TestTrimFuncName(t *testing.T) {
	testTrimFuncName()
}
