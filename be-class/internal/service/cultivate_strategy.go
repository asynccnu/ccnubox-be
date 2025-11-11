package service

import (
	"context"
)

type ToBeStudiedClasses struct {
	IdentityDevelop []ToBeStudiedClass
	SpecificSkill   []ToBeStudiedClass
	CommonEducate   []ToBeStudiedClass
}

type ToBeStudiedClass struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Credit    string `json:"credit"`
	Property  string `json:"property"`
	Studiable string `json:"studiable"`
	Type      string `json:"type"`
}

type CultivateStrategy interface {
	GetToBeStudiedClass(ctx context.Context, stuId, status string) (ToBeStudiedClasses, error)
}
