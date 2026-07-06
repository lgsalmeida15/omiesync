package omie_config

import (
	"context"
	"fmt"
	"sync"

	"omie-sync-api/internal/apperror"
)

type Service interface {
	GetAll(ctx context.Context) ([]*EndpointConfig, error)
	GetByModulo(ctx context.Context, modulo string) (*EndpointConfig, error)
	Update(ctx context.Context, modulo string, req UpdateRequest, userID string) (*EndpointConfig, error)
	
	// GetCached retorna a config da memória. Se não existir, busca no banco.
	GetCached(ctx context.Context, modulo string) (*EndpointConfig, error)
	// RefreshCache recarrega todas as configs do banco.
	RefreshCache(ctx context.Context) error
}

type service struct {
	repo Repository
	
	cache map[string]*EndpointConfig
	mu    sync.RWMutex
}

func NewService(repo Repository) Service {
	return &service{
		repo:  repo,
		cache: make(map[string]*EndpointConfig),
	}
}

func (s *service) GetAll(ctx context.Context) ([]*EndpointConfig, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetByModulo(ctx context.Context, modulo string) (*EndpointConfig, error) {
	return s.repo.GetByModulo(ctx, modulo)
}

func (s *service) Update(ctx context.Context, modulo string, req UpdateRequest, userID string) (*EndpointConfig, error) {
	if req.EndpointPath == "" || req.Action == "" || req.ArrayField == "" {
		return nil, apperror.Unprocessable("endpoint_path, action e array_field são obrigatórios")
	}

	config, err := s.repo.Update(ctx, modulo, req, userID)
	if err != nil {
		return nil, fmt.Errorf("omie_config.service.Update: %w", err)
	}

	// Atualiza cache
	s.mu.Lock()
	s.cache[modulo] = config
	s.mu.Unlock()

	return config, nil
}

func (s *service) GetCached(ctx context.Context, modulo string) (*EndpointConfig, error) {
	s.mu.RLock()
	c, ok := s.cache[modulo]
	s.mu.RUnlock()

	if ok {
		return c, nil
	}

	// Se não está no cache, busca no banco e salva
	config, err := s.repo.GetByModulo(ctx, modulo)
	if err != nil {
		return nil, fmt.Errorf("omie_config.service.GetCached: %w", err)
	}

	s.mu.Lock()
	s.cache[modulo] = config
	s.mu.Unlock()

	return config, nil
}

func (s *service) RefreshCache(ctx context.Context) error {
	configs, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("RefreshCache: %w", err)
	}

	newCache := make(map[string]*EndpointConfig)
	for _, c := range configs {
		newCache[c.Modulo] = c
	}

	s.mu.Lock()
	s.cache = newCache
	s.mu.Unlock()

	return nil
}
