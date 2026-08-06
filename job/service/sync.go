package service

import (
	"context"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/config"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository/mysql"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository/postgres"
)

type UserInfoSyncService struct {
	userDao     mysql.UserSourceDao
	orderDao    mysql.OrderSourceDao
	fullInfoDao postgres.UserFullInfoDao
	batchSize   int
}

func NewUserInfoSyncService(
	userDao mysql.UserSourceDao,
	orderDao mysql.OrderSourceDao,
	fullInfoDao postgres.UserFullInfoDao,
	batchSize int,
) *UserInfoSyncService {
	if batchSize <= 0 {
		batchSize = 500
	}
	return &UserInfoSyncService{
		userDao:     userDao,
		orderDao:    orderDao,
		fullInfoDao: fullInfoDao,
		batchSize:   batchSize,
	}
}

func DefaultUserInfoSyncService() *UserInfoSyncService {
	batchSize := 500
	if config.Config != nil && config.Config.SyncConfig != nil && config.Config.SyncConfig.BatchSize > 0 {
		batchSize = config.Config.SyncConfig.BatchSize
	}
	return NewUserInfoSyncService(
		mysql.GetUserSourceDao(),
		mysql.GetOrderSourceDao(),
		postgres.GetUserFullInfoDao(),
		batchSize,
	)
}

func (s *UserInfoSyncService) Sync(ctx context.Context) error {
	log.Logger.Infow("user info sync started", "batch_size", s.batchSize)

	var (
		afterID        int64
		totalSuccess   int
		totalFailed    int
		totalProcessed int
		batchNo        int
	)

	for {
		users, err := s.userDao.ListUsersAfterID(ctx, afterID, s.batchSize)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			break
		}
		batchNo++

		userIDs := make([]int64, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
		}

		purchaseStats, err := s.orderDao.ListPurchaseStatsByUserIDs(ctx, userIDs)
		if err != nil {
			return err
		}

		upserts := make([]postgres.UserFullInfoUpsert, 0, len(users))
		for _, user := range users {
			stats := purchaseStats[user.ID]
			upserts = append(upserts, postgres.UserFullInfoUpsert{
				UserID: user.ID,
				Profile: postgres.UserProfile{
					RegisteredAt:           user.CreatedAt.UTC().Unix(),
					Market:                 user.Market,
					KycStatus:              user.KYCStatus,
					TotalFiatDepositUSD:    0, // deposit sync deferred
					FiatDepositCount:       0,
					TotalPurchaseAmountUSD: stats.TotalPayAmount.InexactFloat64(),
					PurchaseCount:          stats.PurchaseCount,
					IsRiskUser:             mysql.IsRiskUser(user.RiskLevel),
				},
			})
		}

		if err := s.fullInfoDao.UpsertBatch(ctx, upserts); err != nil {
			totalFailed += len(upserts)
			log.Logger.Errorw("user info sync batch failed", "batch", batchNo, "error", err, "count", len(upserts))
		} else {
			totalSuccess += len(upserts)
		}
		totalProcessed += len(users)
		afterID = users[len(users)-1].ID

		log.Logger.Infow(
			"user info sync batch done",
			"batch", batchNo,
			"after_id", afterID,
			"batch_count", len(users),
			"processed", totalProcessed,
		)

		if len(users) < s.batchSize {
			break
		}
	}

	log.Logger.Infow(
		"user info sync finished",
		"batches", batchNo,
		"processed", totalProcessed,
		"success", totalSuccess,
		"failed", totalFailed,
	)
	if totalFailed > 0 {
		return errSyncPartialFailure
	}
	return nil
}

var errSyncPartialFailure = errString("user info sync completed with failures")

type errString string

func (e errString) Error() string { return string(e) }
