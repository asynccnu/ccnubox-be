# be-class

课程服务的统一架构版本。目录和启动方式与 `be-classlist` 保持一致：

- `main.go`：应用生命周期
- `wire.go`：依赖注入
- `conf/`、`ioc/`：公共配置与基础设施初始化
- `biz/model/`、`biz/usecase/`：领域模型和业务用例
- `repository/`：MySQL、Redis、Elasticsearch 存储
- `service/`：应用服务
- `grpc/`、`http/`：协议适配层
- `cron/`：定时任务

本地运行前，将 `config/config-example.yaml` 和根目录的 `config-infra-example.yaml`
分别复制为 `config/config.yaml` 和 `config/config-infra.yaml`。

```bash
go run .
```
