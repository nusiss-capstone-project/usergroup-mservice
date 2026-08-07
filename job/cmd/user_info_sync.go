package main

import (
	"context"
	"os"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/config"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/service"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/telemetry"
)

func main() {
	config.Init()
	log.InitLogger()
	shutdown := telemetry.Init(context.Background())
	defer func() {
		_ = shutdown(context.Background())
	}()
	repository.Init()

	ctx, end := telemetry.StartRootSpan(context.Background(), "user_info_sync")
	defer end()

	log.WithContext(ctx).Infow("usergroup user_info_sync job starting")
	if err := service.DefaultUserInfoSyncService().Sync(ctx); err != nil {
		log.WithContext(ctx).Errorw("usergroup user_info_sync job failed", "error", err)
		os.Exit(1)
	}
	log.WithContext(ctx).Infow("usergroup user_info_sync job completed")
}
