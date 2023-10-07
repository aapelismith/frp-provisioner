package server

import (
	"context"
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/service"
)

// Server frp controller server
type Server struct {
	srv  *service.Service
	opts *config.FrpOptions
}

// Run start the frp controller server
func (s *Server) Run(ctx context.Context) error {
	return nil
}

// Shutdown the frp controller server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

// New create frp controller server
func New(ctx context.Context, opts *config.FrpOptions) (*Server, error) {
	return nil, nil
}
