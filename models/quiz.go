package models

import (
    "gorm.io/gorm"
)

type QuizStatus string

const (
    QuizStatusDraft     QuizStatus = "draft"
    QuizStatusPublished QuizStatus = "published"
)

type Quiz struct {
    gorm.Model
    Title       string `gorm:"not null"`
    Description string
    Status      QuizStatus `gorm:"type:varchar(20);default:'draft'"`
    UserID      uint       `gorm:"not null"`
    User        User       `gorm:"foreignKey:UserID"`
    Questions   []Question `gorm:"foreignKey:QuizID;constraint:OnDelete:CASCADE"`
    TokenCost   int        `gorm:"default:0"`
    FileType    string    
    FileName    string     
    ShareCode   string     `gorm:"uniqueIndex"` 
}

type QuestionType string

const (
    QuestionTypeMultipleChoice QuestionType = "multiple_choice"
    QuestionTypeTrueFalse      QuestionType = "true_false"
    QuestionTypeShortAnswer    QuestionType = "short_answer"
)

type Question struct {
    gorm.Model
    Content     string       `gorm:"not null"`
    Type        QuestionType `gorm:"type:varchar(20);default:'multiple_choice'"`
    QuizID      uint         `gorm:"not null"`
    Options     []Option     `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE"`
    Explanation string       
}

type Option struct {
    gorm.Model
    Content     string `gorm:"not null"`
    IsCorrect   bool   `gorm:"default:false"`
    QuestionID  uint   `gorm:"not null"`
}