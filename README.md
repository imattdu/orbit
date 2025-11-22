# orbit

**orbit** 是一组为 Go
服务构建的基础设施工具集，包含日志、追踪、上下文管理、HTTP
客户端、统一错误框架等常用组件，旨在提供稳定、可观测、易扩展的微服务基础能力。

本项目遵循「轻量、清晰、可组合」的设计理念，每个模块均为独立
package，可按需使用，也可以组合构建服务内核。

## ✨ Features

### 🌈 Logging (`logx`)

-   多级别日志（Info / Warn / Error）
-   支持按小时、天、大小轮转
-   支持 Info / Warn 分文件写入
-   支持 JSON / Console 输出
-   支持 trace_id 注入
-   支持软链接 `app.log` 指向当前最新文件
-   高性能、低锁、并发安全

### 🌐 HTTP Client (`httpclient`)

-   基于 `net/http` 封装
-   支持重试（固定次数、指数退避）
-   Before / After Hook
-   支持连接超时、读写超时、Per-request 超时
-   自动注入 trace_id
-   可获取重试次数、耗时、响应元数据

### 🧩 Error Framework (`errorx`)

-   标准错误结构：Code、Message、Detail
-   内建常用错误分类（Biz / Sys / Service / Default）
-   支持 Wrap / Unwrap，并携带上下文字段

### 🌀 Context Wrapper (`cctx`)

-   上下文扩展能力（不可变 bag）
-   自动携带 trace_id
-   可在 logger / httpclient / tracex 之间全链路传递

### 🔍 Tracing (`tracex`)

-   全链路 trace_id 管理
-   从 HTTP Header 自动提取
-   支持自定义 trace_id 注入器
-   与 logger 和 httpclient 无缝协作

## 📦 Installation

``` bash
go get github.com/imattdu/orbit
```

## 📁 Project Structure

    orbit/
    │
    ├── cctx/           # Context 扩展（bag / trace_id）
    ├── errorx/         # 统一错误框架
    ├── httpclient/     # HTTP 客户端（重试、超时、Hook）
    ├── logx/           # 高性能日志库
    ├── tracex/         # trace_id 工具
    │
    ├── cmd/
    │   └── demo/       # 示例
    │
    └── README.md

## 🚀 Quick Start Examples

### 1. Logger 使用

``` go
import "github.com/imattdu/orbit/logx"

func main() {
    logger := logx.New(
        logx.WithLevel(logx.LevelInfo),
        logx.WithRotateMode(logx.RotateHourly),
        logx.WithJSON(true),
        logx.WithFilename("./logs/app.log"),
    )

    logger.Info("service started",
        logx.Field("version", "1.0.0"),
    )

    logger.Warn("something may be wrong")
}
```

### 2. HTTP Client 使用

``` go
import (
    "time"
    "github.com/imattdu/orbit/httpclient"
)

func main() {
    cli := httpclient.NewClient(
        httpclient.WithRetry(3),
        httpclient.WithTimeout(3*time.Second),
    )

    resp, err := cli.Get("https://example.com")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
}
```

### 3. Tracing + cctx 使用

``` go
ctx := context.Background()
ctx = tracex.WithTraceID(ctx)

logx.FromContext(ctx).Info("processing",
    logx.Field("trace_id", tracex.GetTraceID(ctx)),
)
```

### 4. Errorx 使用

``` go
import "github.com/imattdu/orbit/errorx"

func doBiz() error {
    return errorx.Wrap(errorx.ExternalErrBiz,
        "业务处理失败",
        errors.New("order not found"),
    )
}
```

## ⚙️ Configuration Reference

### logx

  配置项                      描述
  --------------------------- -----------------------------------------
  `WithLevel(level)`          设置日志级别
  `WithFilename(path)`        设置日志文件路径
  `WithRotateMode(mode)`      RotateDaily / RotateHourly / RotateSize
  `WithMaxSize(bytes)`        按大小轮转
  `WithJSON(bool)`            是否启用 JSON 输出
  `WithTraceIDInjector(fn)`   自定义 trace_id 生成器

### httpclient

  配置项                 描述
  ---------------------- -------------
  `WithRetry(n)`         重试次数
  `WithTimeout(d)`       全局超时
  `WithBeforeHook(fn)`   请求前 Hook
  `WithAfterHook(fn)`    请求后 Hook

## 🧪 Example Project

``` bash
go run ./cmd/demo
```

## 🧑‍💻 Contributing

欢迎 PR 和 Issue。

1.  Fork 本仓库\
2.  创建功能分支：`git checkout -b feature/xxx`\
3.  提交代码并创建 PR\
4.  通过审核即可合并

## 📜 License

Orbit 采用 Apache 2.0 License。详见 LICENSE。

## ❤️ Thanks

感谢所有使用 orbit 构建服务的开发者。如有更多需求，欢迎提出 Issue。
