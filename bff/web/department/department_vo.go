package department

type SaveDepartmentRequest struct {
	Id    int64  `json:"id,omitempty"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Place string `json:"place"`
	Time  string `json:"time"`
}

type Department struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Place string `json:"place"`
	Time  string `json:"time"`
}

type DelDepartmentRequest struct {
	Id int64 `json:"id"`
}

type GetDepartmentsResponse struct {
	Departments []*Department `json:"departments"`
}
