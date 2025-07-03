# Muxi_Library(图书馆服务)

## 一、如何运行？

### 1、配置信息
将`configs/config-example.yaml`换成`configs/config.yaml`,并填充配置文件
### 2、运行
在`be-library\cmd\be-library`下执行`go run .`


## 二、错误码

| 错误码 | 含义                         |
|-----| ---------------------------- |
| 456 | 爬取座位失败                 |
| 457 | 请求user登录服务错误   |

## 三、API文档
将文件中`openapi.yaml`导入到`apifox`中即可 