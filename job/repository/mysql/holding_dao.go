package mysql

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"gorm.io/gorm"
)

// HoldingSourceDao reads user_asset_holdings.
// Current profile sync does not map holdings fields; kept for TD source coverage.
type HoldingSourceDao interface {
	ListUserIDsWithHoldings(ctx context.Context) ([]int64, error)
}

type HoldingSourceDaoImpl struct {
	db *gorm.DB
}

var (
	holdingSourceOnce sync.Once
	holdingSourceDao  *HoldingSourceDaoImpl
)

func GetHoldingSourceDao() *HoldingSourceDaoImpl {
	holdingSourceOnce.Do(func() {
		holdingSourceDao = &HoldingSourceDaoImpl{db: repository.AssetDB}
	})
	return holdingSourceDao
}

func (d *HoldingSourceDaoImpl) ListUserIDsWithHoldings(ctx context.Context) ([]int64, error) {
	var userIDs []int64
	ret := d.db.WithContext(ctx).
		Table("user_asset_holdings").
		Distinct("user_id").
		Pluck("user_id", &userIDs)
	if ret.Error != nil {
		log.Logger.Errorw("failed to list user_asset_holdings user ids", "error", ret.Error)
		return nil, ret.Error
	}
	return userIDs, nil
}
