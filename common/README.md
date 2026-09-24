# common 模块说明

## 概述
`common` 是 CCNUBOX 各业务服务共享的基础依赖层，提供统一的 **Proto 规范、业务中台封装、通用库与工具函数**，以减少重复造轮子、保证观测与配置的一致性。本模块可以独立作为 Go Module 引入，也可通过 `go.work` 在单仓库内直接引用。

## 目录结构
- [`common/api`](common/api/README.md) — gRPC/HTTP Proto 定义与生成产物，集中管理跨服务契约。
- [`common/bizpkg`](common/bizpkg/conf/conf.go:1) — 面向业务的基础设施封装（配置、日志、OTel、GRPC Server、数据源等）。
- [`common/pkg`](common/pkg/cronx/cron.go:1) — 轻量工具库集合，聚焦在单一能力（例如 Cron、限流、Nacos、Prometheus 等）。
- [`common/tool`](common/tool/studentid.go:1) — 简易业务工具函数，覆盖学号解析、学年推断等 CCNU 业务规则。

## 快速开始
1. 在业务模块中引入：
   ```go
   require github.com/asynccnu/ccnubox-be/common vX.Y.Z
   ```
2. 若使用 mono-repo 开发，可在根目录 `go.work` 中 `use ./common`，避免本地反复 `replace`。
3. Proto 相关能力需要本地安装编译器插件（详见 [`common/api/README.md`](common/api/README.md)）：
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   go install github.com/go-errors/errors/cmd/protoc-gen-go-errors@latest
   ```

## 关键能力
| 领域 | 组件 | 说明 |
| --- | --- | --- |
| 配置管理 | [`common/bizpkg/conf`](common/bizpkg/conf/conf.go:1) | `InitConfig` 先尝试 Nacos 拉取，失败时回落本地文件，支持 `mapstructure` 严格校验。|
| 服务注册/治理 | [`common/pkg/grpcx`](common/pkg/grpcx/grpc_server.go:1) | 提供原生 gRPC Server 与 Kratos Server 两套启动器，内置 etcd 注册、加权元数据。|
| 日志与链路 | [`common/pkg/logger/zapx`](common/pkg/logger/zapx/logger.go:1) & [`common/pkg/otelx`](common/pkg/otelx/otel.go:1) | 基于 Zap 封装结构化日志，额外适配 Kratos/GORM；OTel 模块统一 trace/exporter 初始化。|
| 错误处理 | [`common/pkg/errorx`](common/pkg/errorx/error.go:1) | 自带堆栈保留、`Wrap`/`Format` 能力，`%+v` 输出完整链路。|
| 调度与限流 | [`common/pkg/cronx`](common/pkg/cronx/cron.go:1)、[`common/pkg/limiter`](common/pkg/limiter/redis_slide_window.go:1) | cron 管理器支持任务增删查与统一日志；限流器实现 Redis 滑动窗口脚本封装。|
| 配置解析 | [`common/pkg/viperx`](common/pkg/viperx/viper.go:1) | 基于 `mapstructure` 标签递归校验缺失项，减少部署踩坑。|
| 外部服务适配 | [`common/pkg/nacosx`](common/pkg/nacosx/nacos.go:1)、[`common/pkg/qiniu`](common/pkg/qiniu/qiniu.go:1)、[`common/pkg/prometheusx`](common/pkg/prometheusx/prometheus.go:1) | 针对常见外部依赖封装初始化函数，降低重复样板。|
| 业务工具 | [`common/tool/studentid.go`](common/tool/studentid.go:1)、[`common/tool/year_and_semester.go`](common/tool/year_and_semester.go:1) | 解析学号类型、判断毕业、推算当前学年学期等本校特定逻辑。|

## 使用示例
### 统一 Cron 管理
```go
package main

import (
    "context"
    "time"

    "github.com/asynccnu/ccnubox-be/common/pkg/cronx"
    "github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
    "go.uber.org/zap"
)

func main() {
    baseLogger, _ := zap.NewProduction()
    mgr := cronx.NewManager(zapx.NewZapLogger(baseLogger))
    _ = mgr.AddTask("refresh-cache", "@every 5m", func(ctx context.Context, log cronx.Logger) {
        log.Info("start refresh")
        time.Sleep(time.Second)
        log.Info("done")
    })
    defer mgr.Stop(context.Background())
    select {}
}
```

### 从 Nacos 拉取配置并严格校验
```go
package confdemo

import (
    _ "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
    "github.com/asynccnu/ccnubox-be/common/pkg/viperx"
)

type RedisConf struct {
    Addr string `mapstructure:"addr"`
    DB   int    `mapstructure:"db"`
}

func Load() RedisConf {
    var cfg RedisConf
    if err := viperx.MustUnmarshall("redis", &cfg); err != nil {
        panic(err)
    }
    return cfg
}
```

### 学号辅助函数
```go
package rules

import (
    "github.com/asynccnu/ccnubox-be/common/tool"
)

func IsOldStudent(id string) bool {
    return tool.IsGraduated(id)
}
```

## 最佳实践
1. **模块化引用**：对业务层暴露接口时，优先返回 `common/pkg/errorx` 中的错误，以便统一日志/告警串联。
2. **观测一致性**：所有长生命周期任务、gRPC Server 请同时注入 [`common/pkg/otelx`](common/pkg/otelx/otel.go:1) 和 [`common/pkg/logger`](common/pkg/logger/types.go:1) 产出的实例，保证 Trace/Log 关联。
3. **配置兜底**：上线前确保对应 Nacos DSN 已配置，同时在镜像内附带 `config*.yaml`，避免远程拉取失败导致 panic。
4. **代码生成**：Proto 变更后在 [`common/api`](common/api/README.md) 根目录执行 `make`（或手动 `protoc`），然后同步 go.mod 依赖。

## 贡献指引
1. 新增通用能力时，请区分是否属于业务中台（放 `bizpkg`）或通用库（放 `pkg`）。
2. 所有公共 API 需配套最小使用示例以及 `godoc` 注释。
3. 变更 Proto 需同步 bump `common` 版本并在相应服务中验证兼容性。

## Kafka 消费失败处理

### 失败分类

消费失败分两类，处理方式不同：

| 类型 | 判定方式 | 处理 |
| --- | --- | --- |
| 依赖故障（DB、网络、下游） | 普通错误 | 保留位点重复重投，直到依赖恢复，永不被跳过 |
| 永久失败（消息体非法、字段非法等） | `saramax.Permanent(err)` 包装 | 阈值内保留位点等人工介入，超过阈值后跳过该消息 |

跳过阈值默认 5 次（`saramax.DefaultSkipAttempts`），可用 `saramax.HandlerConfig.SkipAttempts` 调整，
传 0 用默认值，传负数彻底禁用跳过。批量消费的整批失败会先逐条重试定位：
正常的消息照常确认，只有确认处理不了的坏消息才进入跳过计数；
整批一条都没成功时判为依赖整体不可用，不跳过任何消息。

### 坏消息与人工恢复

无 DLQ 模式下，坏消息在达到跳过阈值前会阻塞对应分区，并且会让整个会话结束：
同一组内其他分区会立刻停止并等待下一次会话重建，这些分区只能按退避间隔获得很短的处理窗口。
日志提供 `topic`、`partition`、`offset`，不打印包含隐私的消息体。

1. 根据日志定位原始消息，在受控环境读取并按需安全存档；当前实现不提供独立的失败消息存档。
2. 修复生产者、消费兼容逻辑或依赖服务，然后恢复消费。只修复生产者不会改变已经写入 Kafka 的坏消息。
3. 被跳过的消息只有 `KAFKA_MESSAGE_DROPPED` 日志和 metrics，没有存档，只能靠日志定位后从业务侧确认或补偿。
4. Kafka 未提交位点不等于永久保存消息。故障期间需检查消费积压及 topic 保留时间/容量，在保留期限前处理或存档，避免消息被清理。

### panic 兜底

业务处理 panic 时 `saramax.Catch` 会就地恢复，转成一次普通处理失败（不确认位点、按重试策略处理），
不会崩掉 sarama 的 claim 协程和整个进程。消费逻辑 panic 不代表消息可以丢弃，
因此 panic 不参与跳过计数，反复 panic 的消息仍然需要人工介入。

### 日志关键字

不新增 metrics 的前提下，故障发现依赖日志。所有“需要人工介入”和“消息被丢弃”的日志都带有固定关键字，建议直接在日志平台按关键字检索或告警：

| 关键字 | 含义 |
| --- | --- |
| `KAFKA_PARTITION_BLOCKED` | 位点未确认且分区停止消费（坏消息未达跳过阈值、依赖长期不可用、classlist 原地重试达到 10 次），必须人工处理 |
| `KAFKA_CONSUME_RETRY` | 本轮处理失败、正在退避重试，通常可自愈；持续出现说明依赖不稳定 |
| `KAFKA_MESSAGE_DROPPED` | 消息被显式丢弃，不会再重投（永久失败达到跳过阈值、延迟消息过期、无法入库的木犀计划） |
| `KAFKA_SEND_FAILED` | 消息没有进入 Kafka（发送失败、冷却期内被拒、群发中断），需要按幂等 key 补发 |
| `KAFKA_CONSUMER_PANIC` | 消费逻辑 panic，已兜底成一次处理失败；反复出现说明代码或数据有 bug，需要人工介入 |

`KAFKA_PARTITION_BLOCKED` 出现即代表对应分区在人工处理前不会前进，建议设为告警。
`KAFKA_MESSAGE_DROPPED` 代表消息已经丢失，需要结合业务确认能否补偿。
