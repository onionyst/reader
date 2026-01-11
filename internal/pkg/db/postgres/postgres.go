package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     int
	DBName   string
	User     string
	Password string

	SSLMode  string
	TimeZone string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	ConnectTimeout time.Duration

	LogLevel logger.LogLevel
}

func (c Config) DSN() string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.TimeZone == "" {
		c.TimeZone = "UTC"
	}

	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s TimeZone=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode, c.TimeZone)
}

func Close(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func FromEnv() (Config, error) {
	portStr := os.Getenv("POSTGRES_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid POSTGRES_PORT=%q: %w", portStr, err)
	}

	cfg := Config{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     port,
		DBName:   os.Getenv("POSTGRES_DB"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		TimeZone: "UTC",
	}

	return cfg, nil
}

func Open(cfg Config) (*gorm.DB, error) {
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 20
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 5 * time.Minute
	}
	if cfg.LogLevel == 0 {
		cfg.LogLevel = logger.Silent
	}

	dsn := cfg.DSN()

	gdb, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return gdb, nil
}
