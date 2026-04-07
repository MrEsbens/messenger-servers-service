package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	serversv1 "github.com/MrEsbens/messenger-servers-service/api/servers/v1"
)

type Server struct {
	grpcServer *grpc.Server
	port       string
}

func NewServer(port string, handler *Handler) *Server {
	grpcServer := grpc.NewServer(
		// Keepalive параметры для стабильности
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              30 * time.Second,
			Timeout:           5 * time.Second,
			MaxConnectionAge:  10 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		// Максимальный размер сообщения (10MB)
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	// Регистрируем handler
	serversv1.RegisterServersServiceServer(grpcServer, handler)

	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
	}

	go func() {
		<-ctx.Done()
		s.grpcServer.GracefulStop()
	}()

	fmt.Printf("🚀 gRPC server started on port %s\n", s.port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
