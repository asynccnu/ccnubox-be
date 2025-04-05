# 📘 Grade 服务接口文档

该接口文档定义了与成绩查询相关的接口，包括查询某学期成绩、查询成绩类型等功能。

## 🍪 Grade 服务接口

### 1. 获取某学期成绩

- **接口名称**：`GetGradeByTerm`
- **调用方式**：RPC（gRPC）
- **请求路径**：`grade.v1.GradeService/GetGradeByTerm`
- **功能描述**：根据学号和学年学期获取该学期的所有课程成绩。

#### ✅ 请求参数（GetGradeByTermReq）

```
{
  "studentId": "2023123456",
  "xnm": 2024,
  "xqm": 1
}
```

- `studentId`：学号。
- `xnm`：学年名，表示学年，如 2024 表示 2024-2025 学年。
- `xqm`：学期名，表示学期，如 1 表示第一学期。

#### 📦 响应参数（GetGradeByTermResp）

```
{
  "grades": [
    {
      "Kcmc": "数学分析",
      "Xf": 4.0,
      "Cj": 90.5,
      "kcxzmc": "专业主干课程",
      "Kclbmc": "专业课",
      "kcbj": "主修",
      "jd": 4.0,
      "regularGradePercent": "30%",
      "regularGrade": 85.0,
      "finalGradePercent": "70%",
      "finalGrade": 95.0
    },
    {
      "Kcmc": "计算机导论",
      "Xf": 3.0,
      "Cj": 88.0,
      "kcxzmc": "通识必修课",
      "Kclbmc": "公共课",
      "kcbj": "主修",
      "jd": 3.7,
      "regularGradePercent": "40%",
      "regularGrade": 80.0,
      "finalGradePercent": "60%",
      "finalGrade": 92.0
    }
  ]
}
```

### 2. 获取成绩类型

- **接口名称**：`GetGradeScore`
- **调用方式**：RPC（gRPC）
- **请求路径**：`grade.v1.GradeService/GetGradeScore`
- **功能描述**：获取某个学生所有课程的成绩类型。

#### ✅ 请求参数（GetGradeScoreReq）

```
{
  "studentId": "2023123456"
}
```

#### 📦 响应参数（GetGradeScoreResp）

```
{
  "typeOfGradeScore": [
    {
      "kcxzmc": "专业主干课程",
      "gradeScoreList": [
        {
          "Kcmc": "数学分析",
          "Xf": 4.0
        },
        {
          "Kcmc": "高等数学",
          "Xf": 3.5
        }
      ]
    },
    {
      "kcxzmc": "通识必修课",
      "gradeScoreList": [
        {
          "Kcmc": "计算机导论",
          "Xf": 3.0
        }
      ]
    }
  ]
}
```

## 🔗 涉及下游调用服务

- `be-user`
- `classList`
- `be-feed`
