package main

import (
	"context"
	"os"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/config"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/repository"
	"github.com/nusiss-capstone-project/usergroup-mservice/job/service"
)

func main() {
	config.Init()
	log.InitLogger()
	repository.Init()

	log.Logger.Infow("usergroup user_info_sync job starting")
	if err := service.DefaultUserInfoSyncService().Sync(context.Background()); err != nil {
		log.Logger.Errorw("usergroup user_info_sync job failed", "error", err)
		os.Exit(1)
	}
	log.Logger.Infow("usergroup user_info_sync job completed")
}
