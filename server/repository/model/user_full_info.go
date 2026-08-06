package model

import (
	"time"

	"gorm.io/datatypes"
)

type UserFullInfo struct {
	ID        int64          `gorm:"primaryKey;autoIncrement"`
	UserID    int64          `gorm:"not null;uniqueIndex:uk_user_full_info_user_id"`
	Profile   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null"`
}

func (UserFullInfo) TableName() string {
	return "user_full_info"
}
