package service

import (
	"github.com/babylonchain/babylon-finality-gadget/service/rpc"

	"github.com/babylonchain/babylon-finality-gadget/config"
)

type Service struct {
	cfg       *config.Config
	rpcServer *rpc.Server
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg:       cfg,
		rpcServer: rpc.NewServer(cfg.RpcListener),
	}
}

func (s *Service) Start() error {
	return s.rpcServer.Start()
}
