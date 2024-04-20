package grpc

import (
	"fmt"
	"net"

	g "github.com/apple5343/golangProjectV2/internal/grpc"
	"google.golang.org/grpc"
)

type App struct {
	gRPCServer *grpc.Server
	Port       int
}

func New(calcService g.Calc, authSevice g.Auth, port int) *App {
	gRPCServer := grpc.NewServer()
	g.Register(gRPCServer, calcService, authSevice)
	return &App{
		gRPCServer: gRPCServer,
		Port:       port,
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.Port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	a.gRPCServer.GracefulStop()
}
