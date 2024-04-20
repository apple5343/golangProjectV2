package app

import (
	"fmt"
	"net/http"

	g "github.com/apple5343/golangProjectV2/internal/app/grpc"
	"github.com/apple5343/golangProjectV2/internal/app/websocket"
	"github.com/apple5343/golangProjectV2/internal/config"
	"github.com/apple5343/golangProjectV2/internal/services/auth"
	"github.com/apple5343/golangProjectV2/internal/services/calculator"
	storage "github.com/apple5343/golangProjectV2/internal/storage/sqlite"
	c "github.com/apple5343/grpc"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	db         *storage.SqlDB
	router     *mux.Router
	config     *config.Config
	manager    *websocket.Manager
	GRPCServer *g.App
	auth       c.AuthClient
	calculator c.CalculatorClient
}

func New(storagePath string, cfg *config.Config) (*Server, error) {
	router := mux.NewRouter()
	db, err := storage.OpenStorage(cfg.StoragePath)
	if err != nil {
		return &Server{}, err
	}
	manager := websocket.NewManager(cfg.SecretJWT)
	calculator, err := calculator.NewCalculator(cfg, *db, manager.ListenUpdatesCh)
	if err != nil {
		return &Server{}, err
	}
	auth := auth.New(db, cfg.SecretJWT, cfg.TokenTTL)
	grpcapp := g.New(calculator, auth, cfg.GRPC.Port)
	conn, err := grpc.Dial(fmt.Sprintf(":%d", grpcapp.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return &Server{}, err
	}
	authClient := c.NewAuthClient(conn)
	calcClient := c.NewCalculatorClient(conn)
	s := &Server{db: db, config: cfg, router: router, GRPCServer: grpcapp, auth: authClient, calculator: calcClient, manager: manager}
	calculator.ContinueCalculations()
	s.SetupRoutes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
