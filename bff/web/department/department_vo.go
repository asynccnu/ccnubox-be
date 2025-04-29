package department

type SaveDepartmentRequest struct {
	Id    int64  `json:"id"` //可选,如果新增记录不用填写
	Name  string `json:"name" binding:"required"`
	Phone string `json:"phone" binding:"required"`
	Place string `json:"place" binding:"required"`
	Time  string `json:"time" binding:"required"`
}

type Department struct {
	Id    int64  `json:"id" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Phone string `json:"phone" binding:"required"`
	Place string `json:"place" binding:"required"`
	Time  string `json:"time" binding:"required"`
}

type DelDepartmentRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetDepartmentsResponse struct {
	Departments []*Department `json:"departments" binding:"required"`
}
