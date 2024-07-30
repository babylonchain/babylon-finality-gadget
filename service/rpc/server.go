package rpc

import (
	"fmt"
	"net"

	"github.com/babylonchain/babylon-finality-gadget/proto"
	"google.golang.org/grpc"
)

type Server struct {
	rpcListener string
	grpcServer  *grpc.Server
	proto.UnimplementedFinalityGadgetServer
}

func NewServer(rpcListener string) *Server {
	grpcServer := grpc.NewServer()
	return &Server{
		rpcListener: rpcListener,
		grpcServer:  grpcServer,
	}
}

func (s *Server) Start() error {
	proto.RegisterFinalityGadgetServer(s.grpcServer, s)
	listener, err := net.Listen("tcp", s.rpcListener)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.rpcListener, err)
	}
	defer listener.Close()
	_ = s.grpcServer.Serve(listener)
	return nil
}
