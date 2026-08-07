package mysql

import (
	"context"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"gorm.io/gorm"
)

const riskLevelHigh = "HIGH"

type IdentityUser struct {
	ID        int64     `gorm:"column:id"`
	Market    string    `gorm:"column:market"`
	KYCStatus string    `gorm:"column:kyc_status"`
	RiskLevel string    `gorm:"column:risk_level"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (IdentityUser) TableName() string {
	return "users"
}

type UserSourceDao interface {
	// ListUsersAfterID returns users with id > afterID ordered by id ASC, limited to limit.
	ListUsersAfterID(ctx context.Context, afterID int64, limit int) ([]IdentityUser, error)
}

type UserSourceDaoImpl struct {
	db *gorm.DB
}

var (
	userSourceOnce sync.Once
	userSourceDao  *UserSourceDaoImpl
)

func GetUserSourceDao() *UserSourceDaoImpl {
	userSourceOnce.Do(func() {
		userSourceDao = &UserSourceDaoImpl{db: repository.IdentityDB}
	})
	return userSourceDao
}

func (d *UserSourceDaoImpl) ListUsersAfterID(ctx context.Context, afterID int64, limit int) ([]IdentityUser, error) {
	if limit <= 0 {
		limit = 500
	}
	var users []IdentityUser
	ret := d.db.WithContext(ctx).
		Select("id", "market", "kyc_status", "risk_level", "created_at").
		Where("id > ?", afterID).
		Order("id ASC").
		Limit(limit).
		Find(&users)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to list identity users batch", "error", ret.Error, "after_id", afterID, "limit", limit)
		return nil, ret.Error
	}
	return users, nil
}

func IsRiskUser(riskLevel string) bool {
	return riskLevel == riskLevelHigh
}
