package service

import (
	"go.uber.org/zap"
)

type Service struct {
	log *zap.Logger
}

func NewService(log *zap.Logger) *Service {
	return &Service{log: log}
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}
