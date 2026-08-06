package dao

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/usergroup-mservice/server/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/server/repository"
	"gorm.io/gorm"
)

type UserFullInfoDao interface {
	CountByExpression(ctx context.Context, expression string) (int64, error)
	ExistsByUserAndExpression(ctx context.Context, userID int64, expression string) (bool, error)
}

type UserFullInfoDaoImpl struct {
	db *gorm.DB
}

var (
	userFullInfoOnce sync.Once
	userFullInfoDao  *UserFullInfoDaoImpl
)

func GetUserFullInfoDao() *UserFullInfoDaoImpl {
	userFullInfoOnce.Do(func() {
		userFullInfoDao = &UserFullInfoDaoImpl{db: repository.DB}
	})
	return userFullInfoDao
}

func (d *UserFullInfoDaoImpl) CountByExpression(ctx context.Context, expression string) (int64, error) {
	var count int64
	ret := d.db.WithContext(ctx).
		Raw(`SELECT COUNT(*) FROM user_full_info WHERE profile @@ CAST(? AS jsonpath)`, expression).
		Scan(&count)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to count user_full_info by expression", "error", ret.Error)
		return 0, ret.Error
	}
	return count, nil
}

func (d *UserFullInfoDaoImpl) ExistsByUserAndExpression(ctx context.Context, userID int64, expression string) (bool, error) {
	var matched bool
	ret := d.db.WithContext(ctx).
		Raw(
			`SELECT EXISTS(
				SELECT 1 FROM user_full_info
				WHERE user_id = ? AND profile @@ CAST(? AS jsonpath)
			)`,
			userID, expression,
		).
		Scan(&matched)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw(
			"failed to match user_full_info by expression",
			"error", ret.Error,
			"user_id", userID,
		)
		return false, ret.Error
	}
	return matched, nil
}
