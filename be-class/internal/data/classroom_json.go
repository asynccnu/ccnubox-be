package data

type ClassroomJSONData struct{}

func NewClassroomJSONData() *ClassroomJSONData {
	return &ClassroomJSONData{}
}

func (d *ClassroomJSONData) ClassroomJSON() []byte {
	classrooms := make([]byte, len(classListBytes))
	copy(classrooms, classListBytes)
	return classrooms
}
