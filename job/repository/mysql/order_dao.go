package mysql

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const orderStatusPaySucceed = "pay_succeed"

type PurchaseStats struct {
	UserID         int64           `gorm:"column:user_id"`
	PurchaseCount  int64           `gorm:"column:purchase_count"`
	TotalPayAmount decimal.Decimal `gorm:"column:total_pay_amount"`
}

type OrderSourceDao interface {
	ListPurchaseStatsByUserIDs(ctx context.Context, userIDs []int64) (map[int64]PurchaseStats, error)
}

type OrderSourceDaoImpl struct {
	db *gorm.DB
}

var (
	orderSourceOnce sync.Once
	orderSourceDao  *OrderSourceDaoImpl
)

func GetOrderSourceDao() *OrderSourceDaoImpl {
	orderSourceOnce.Do(func() {
		orderSourceDao = &OrderSourceDaoImpl{db: repository.AssetDB}
	})
	return orderSourceDao
}

func (d *OrderSourceDaoImpl) ListPurchaseStatsByUserIDs(ctx context.Context, userIDs []int64) (map[int64]PurchaseStats, error) {
	out := make(map[int64]PurchaseStats, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []PurchaseStats
	ret := d.db.WithContext(ctx).
		Table("asset_orders").
		Select("user_id, COUNT(*) AS purchase_count, COALESCE(SUM(pay_amount), 0) AS total_pay_amount").
		Where("status = ? AND user_id IN ?", orderStatusPaySucceed, userIDs).
		Group("user_id").
		Find(&rows)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("failed to list purchase stats batch", "error", ret.Error, "user_count", len(userIDs))
		return nil, ret.Error
	}
	for _, row := range rows {
		out[row.UserID] = row
	}
	return out, nil
}
