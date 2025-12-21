# 脚本目录说明

所有脚本均需要在项目**根目录**下执行

下面是对脚本功能的说明：

- `build-{service}.sh`：构建项目的脚本，用于打包成docker镜像，版本均为`v1`，其中`build-all.sh`是一次打包所有镜像，注意`build-be-user.sh`和`build-all.sh`都需要传入加密密钥参数，如果未传入，则使用默认值（注意传入的加密密钥大小应为**16字节 | 24字节 | 32字节**）
- `infra-setup.sh`：一键部署依赖组件脚本，包含`etcd`、`mysql`、`redis`、`kafka`和`ElasticSearch`等基础组件
- `service-setup.sh`：一键部署所有服务脚本，需确保依赖组件已部署完成，同时可传入需单独步数的服务名称参数，多个服务名称用空格隔开
- `sync-config.sh`：同步配置文件脚本，会将所有项目中的样例配置文件copy到`deployment/configs`目录下，`service-setup.sh`需要使用该配置进行部署，这个脚本是为了方便同步配置文件，同时加上参数`-r`可以反向copy
- `update-dependency.sh`：更新依赖脚本，它会到各个目录下执行`go mod tidy`，方便更新所有项目的go模块依赖