package db

import (
	"gorm.io/gorm"

	"reader/internal/pkg/db/postgres"
)

type Connections struct {
	Main *gorm.DB
}

func (c *Connections) Close() {
	_ = postgres.Close(c.Main)
}

func Setup() (*Connections, error) {
	mainCfg, err := postgres.FromEnv()
	if err != nil {
		return nil, err
	}

	mainDB, err := postgres.Open(mainCfg)
	if err != nil {
		return nil, err
	}

	return &Connections{Main: mainDB}, nil
}
