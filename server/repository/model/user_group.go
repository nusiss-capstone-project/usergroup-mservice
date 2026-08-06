package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	UserGroupStatusDraft   = "DRAFT"
	UserGroupStatusActive  = "ACTIVE"
	UserGroupStatusOffline = "OFFLINE"
)

type UserGroup struct {
	ID         int64          `gorm:"primaryKey;autoIncrement"`
	Name       string         `gorm:"type:varchar(255);not null"`
	RuleConfig datatypes.JSON `gorm:"type:jsonb;not null"`
	Expression string         `gorm:"type:text;not null"`
	Status     string         `gorm:"type:varchar(32);not null;index"`
	CreatedAt  time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt  time.Time      `gorm:"type:timestamptz;not null"`
}

func (UserGroup) TableName() string {
	return "user_group"
}
