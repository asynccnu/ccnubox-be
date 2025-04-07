# classService

## 一、如何运行？

### 1、配置信息

将`configs/config-example.yaml`换成`configs/config.yaml`,并填充配置文件
### 2、构建镜像
在`DockerFile`所在目录下使用命令`docker build -t extra_class:v1`构建镜像
>构建镜像，是需要拉取golang:1.22和debian:stable-slim这两个镜像的，当然，如果你是在自己机子上跑，挂个梯子就可以拉取这两个镜像了，但是如果你是在云服务器上拉取的话，很有可能拉取不了（被墙），这是你可以尝试过构建自己的阿里云镜像仓库，然后现在自己的机子上拉取那两个镜像，然后改下tag，上传至自己的阿里云的镜像仓库，然后你的服务器就可以从你自己的阿里云镜像仓库中拉取这两个镜像了
>
>参考教程如下:
>
>[如何构建自己的阿里云镜像仓库](https://blog.csdn.net/qq_26709459/article/details/128726699)


### 3、运行
在`deploy`下执行`docker-compose up -d`即可

## 二、错误码

| 错误码 | 含义 |
|-------|-----|
| 200|成功|
|450|创建classInfo失败|
|451|查询classInfo失败|
|452|查询freeClassroom失败|
|453|CCNU登录失败|

## 三、API文档
将文件中`openapi.yaml`导入到`apifox`中即可

## 四、项目说明

项目依赖于ElasticSearch，课表服务，以及用户服务
项目在启动时，会拉取课表服务的课程信息保存到es，同时会从本地es中来取空闲教室信息到本地另一个索引

注意，该服务额外开启了一个http服务，来上传选课手册
按照代码里面的写法，上传的url在`/class_selection/upload`，当然你也可以自己修改

### UploadSelection API 文档

#### 接口描述
该接口用于上传选课手册 Excel 文件，并解析文件中的上课时间和教学地点信息，存储到数据库。

#### 请求方式
**POST** `/class_selection/upload`

#### 请求头
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Content-Type | `multipart/form-data` | 是 | 表示请求是多部分表单数据 |

#### 请求参数
##### FormData 参数
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| json_data | `string` | 是 | JSON 格式的请求数据，包含学年、学期和表格信息 |
| file | `file` | 是 | Excel 文件，包含选课手册数据 |

##### `json_data` 字段格式
```json
{
  "year": "2024",  
  "semester": "Spring",  
  "sheets": {  
    "Sheet1": {  
      "class_time_idx": 6,  
      "class_where_idx": 7  
    },  
    "Sheet2": {  
      "class_time_idx": 4,  
      "class_where_idx": 5  
    }  
  }  
}
```
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | `string` | 是 | 学年，如 `2024` |
| semester | `string` | 是 | 学期，如 `Spring` |
| sheets | `object` | 是 | 需要解析的表格数据，每个表名对应 `NecessaryIndex` |
| sheets.<sheet_name> | `object` | 是 | 每个 sheet 需要解析的列索引 |
| sheets.<sheet_name>.class_time_idx | `uint` | 是 | 上课时间所在的列索引（从 0 开始） |
| sheets.<sheet_name>.class_where_idx | `uint` | 是 | 教学地点所在的列索引（从 0 开始） |

#### 响应数据
##### 成功响应
```json
{
  "msg": "success"
}
```

##### 失败响应
| HTTP 状态码 | 说明 |
|-------------|------|
| 400 | 请求参数错误，如 JSON 格式错误或文件缺失 |
| 405 | 请求方法错误，非 POST 请求 |
| 500 | 服务器内部错误，解析 Excel 失败或存储失败 |

##### 失败响应示例
```json
{
  "error": "Invalid JSON format"
}
```

#### 注意事项
- `class_time_idx` 和 `class_where_idx` 需确保正确，否则解析可能失败。
- 文件大小不能超过 32MB，否则可能解析失败。
- 仅支持 Excel 格式（`.xlsx`）。


