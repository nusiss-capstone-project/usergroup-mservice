package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFullInfo struct {
	ID        int64          `gorm:"primaryKey;autoIncrement"`
	UserID    int64          `gorm:"column:user_id;uniqueIndex"`
	Profile   datatypes.JSON `gorm:"column:profile;type:jsonb"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
}

func (UserFullInfo) TableName() string {
	return "user_full_info"
}

// UserProfile matches allowedFields in server/service/user_group_service.go.
type UserProfile struct {
	RegisteredAt           int64   `json:"registeredAt"`
	Market                 string  `json:"market"`
	KycStatus              string  `json:"kycStatus"`
	TotalFiatDepositUSD    float64 `json:"totalFiatDepositUSD"`
	FiatDepositCount       int64   `json:"fiatDepositCount"`
	TotalPurchaseAmountUSD float64 `json:"totalPurchaseAmountUSD"`
	PurchaseCount          int64   `json:"purchaseCount"`
	IsRiskUser             bool    `json:"isRiskUser"`
}

type UserFullInfoUpsert struct {
	UserID  int64
	Profile UserProfile
}

type UserFullInfoDao interface {
	UpsertBatch(ctx context.Context, rows []UserFullInfoUpsert) error
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
		userFullInfoDao = &UserFullInfoDaoImpl{db: repository.PostgresDB}
	})
	return userFullInfoDao
}

func (d *UserFullInfoDaoImpl) UpsertBatch(ctx context.Context, rows []UserFullInfoUpsert) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	models := make([]UserFullInfo, 0, len(rows))
	for _, row := range rows {
		raw, err := json.Marshal(row.Profile)
		if err != nil {
			log.Logger.Errorw("failed to marshal user profile", "error", err, "user_id", row.UserID)
			return err
		}
		models = append(models, UserFullInfo{
			UserID:    row.UserID,
			Profile:   datatypes.JSON(raw),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Use Create (not CreateInBatches) + explicit EXCLUDED/NOW so conflict updates always refresh updated_at.
	ret := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"profile":    gorm.Expr("EXCLUDED.profile"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&models)
	if ret.Error != nil {
		log.Logger.Errorw("failed to upsert user_full_info batch", "error", ret.Error, "count", len(models))
		return ret.Error
	}
	log.Logger.Infow(
		"user_full_info batch upserted",
		"count", len(models),
		"rows_affected", ret.RowsAffected,
		"first_user_id", models[0].UserID,
		"last_user_id", models[len(models)-1].UserID,
	)
	return nil
}
