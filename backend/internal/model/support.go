package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey"`
	Username     string    `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	DisplayName  string    `gorm:"size:120;not null"`
	Role         string    `gorm:"size:40;index;not null"`
	Active       bool      `gorm:"index;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (User) TableName() string { return "users" }

type AuditLog struct {
	ID           uint      `gorm:"primaryKey"`
	RequestID    string    `gorm:"size:64;index;not null"`
	ActorID      uint      `gorm:"index;not null"`
	ActorName    string    `gorm:"size:120;not null"`
	Action       string    `gorm:"size:100;index;not null"`
	EntityType   string    `gorm:"size:80;index;not null"`
	EntityID     uint      `gorm:"index;not null"`
	BeforeJSON   string    `gorm:"type:text;not null"`
	AfterJSON    string    `gorm:"type:text;not null"`
	MetadataJSON string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"index;not null"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type Actor struct {
	ID          uint
	Username    string
	DisplayName string
	Role        string
	RequestID   string
}
