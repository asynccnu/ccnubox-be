package service

import "net/http"

type ClassroomJSONProvider interface {
	ClassroomJSON() []byte
}

type ClassroomJSONSvc struct {
	provider ClassroomJSONProvider
}

func NewClassroomJSONSvc(provider ClassroomJSONProvider) *ClassroomJSONSvc {
	return &ClassroomJSONSvc{
		provider: provider,
	}
}

func (s *ClassroomJSONSvc) GetClassrooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.provider.ClassroomJSON())
}
