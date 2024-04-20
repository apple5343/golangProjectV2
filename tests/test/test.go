package test

import (
	"context"
	"log"
	"net"
	"strconv"
	"testing"

	"github.com/apple5343/golangProjectV2/internal/config"
	s "github.com/apple5343/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Test struct {
	*testing.T
	Cfg        *config.Config
	AuthClient s.AuthClient
	CalcClient s.CalculatorClient
}

const (
	grpcHost = "localhost"
)

func New(t *testing.T) (context.Context, *Test) {
	t.Helper()
	t.Parallel()

	cfg, err := config.InitConfig("../config/local.yaml")
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GRPC.Timeout)

	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	cc, err := grpc.DialContext(context.Background(),
		grpcAddress(cfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}

	return ctx, &Test{
		T:          t,
		Cfg:        cfg,
		AuthClient: s.NewAuthClient(cc),
		CalcClient: s.NewCalculatorClient(cc),
	}
}

func grpcAddress(cfg *config.Config) string {
	return net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GRPC.Port))
}
