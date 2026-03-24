package dao

import "gorm.io/gorm"

type DiscussionDAO interface {
}

type discussionDAO struct {
	db *gorm.DB
}

func NewDiscussionDAO(db *gorm.DB) DiscussionDAO {
	return &discussionDAO{db: db}
}
