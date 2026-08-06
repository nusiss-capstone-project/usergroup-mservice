package repository

import (
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	IdentityDB *gorm.DB
	AssetDB    *gorm.DB
	PostgresDB *gorm.DB
)

func Init() {
	identityDSN := os.Getenv("IDENTITY_MYSQL_DSN")
	if identityDSN == "" {
		panic("IDENTITY_MYSQL_DSN environment variable is not set")
	}
	assetDSN := os.Getenv("ASSET_MYSQL_DSN")
	if assetDSN == "" {
		panic("ASSET_MYSQL_DSN environment variable is not set")
	}
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		panic("POSTGRES_DSN environment variable is not set")
	}

	var err error
	IdentityDB, err = gorm.Open(mysql.Open(identityDSN), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err)
	}

	AssetDB, err = gorm.Open(mysql.Open(assetDSN), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err)
	}

	PostgresDB, err = gorm.Open(postgres.Open(postgresDSN), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err)
	}
}
