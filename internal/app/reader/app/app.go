package app

import (
	"reader/internal/app/reader/app/services"
	"reader/internal/app/reader/models"
)

type App struct {
	Repo *models.Repo
	Serv *services.Service
}

func NewApp(repo *models.Repo) *App {
	serv := services.NewService(repo)

	return &App{
		Repo: repo,
		Serv: serv,
	}
}
