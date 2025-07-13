package models

import (
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Id          string `json:"id" gorm:"primaryKey;column:id"`
	PostContent string `json:"postContent" gorm:"column:post_content;not null"`
	UserId      string `json:"userId" gorm:"column:user_id;not null"`
}
