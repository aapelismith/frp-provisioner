package server

import (
	"context"
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/service"
)

// Server frp controller server
type Server struct {
	srv *service.Service
	cfg *config.Configuration
}

// Start the frp controller server
func (s *Server) Start(ctx context.Context) error {
	return nil
}

// New create frp controller server
func New(ctx context.Context, cfg *config.Configuration) (*Server, error) {
	return &Server{
		srv: nil,
		cfg: cfg,
	}, nil
}
