package services

import (
	"reader/internal/app/reader/models"
)

type Service struct {
	repo *models.Repo
}

func NewService(repo *models.Repo) *Service {
	return &Service{
		repo: repo,
	}
}
