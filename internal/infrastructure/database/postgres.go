package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"file-management-service/config"
	"file-management-service/internal/domain/entity"
)

type PostgresDB struct {
	DB *gorm.DB
}

func NewPostgres(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	gormCfg := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: true,
	}

	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	pdb := &PostgresDB{DB: db}
	if err := pdb.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("auto-migrating schema: %w", err)
	}

	return pdb, nil
}

func (p *PostgresDB) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (p *PostgresDB) AutoMigrate() error {
	return p.DB.AutoMigrate(
		&entity.User{},
		&entity.RefreshToken{},
		&entity.Folder{},
		&entity.File{},
		&entity.FileVersion{},
		&entity.FileChunk{},
		&entity.Permission{},
		&entity.ShareLink{},
		&entity.AuditLog{},
		&entity.Notification{},
	)
}

func (p *PostgresDB) Health() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
