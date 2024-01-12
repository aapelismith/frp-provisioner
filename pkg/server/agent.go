package server

import (
	"context"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
)

type AgentServer struct{}

// Start the frp-provisioner controller server
func (s *AgentServer) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting frp-provisioner agent")

	return nil
}

// NewAgentServer create frp-provisioner agent server
func NewAgentServer(ctx context.Context, cfg *config.AgentConfiguration) (*AgentServer, error) {
	return &AgentServer{}, nil
}
