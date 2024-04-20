package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/apple5343/golangProjectV2/internal/services/auth"
	"github.com/apple5343/golangProjectV2/internal/services/calculator"
	storage "github.com/apple5343/golangProjectV2/internal/storage/sqlite"
	c "github.com/apple5343/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Calc interface {
	NewTask(int, string) (*calculator.Task, error)
	GetAllTasks(int64) ([]map[string]interface{}, error)
	GetWorkersInfo() ([]map[string]interface{}, error)
	UpdateDelays(map[string]int) error
	GetDelays() (map[string]int, error)
	GetTaskById(int64, int64) (string, error)
}

type Auth interface {
	Register(name, password string) (int64, error)
	Login(name, password string) (string, error)
	IsAdmin(userID int64) (bool, error)
	GetUserInfo(userID int64) (string, error)
}

type serverAPI struct {
	c.UnimplementedAuthServer
	c.UnimplementedCalculatorServer
	Calc Calc
	Auth Auth
}

func Register(gRPCServer *grpc.Server, calc Calc, auth Auth) {
	c.RegisterAuthServer(gRPCServer, &serverAPI{Calc: calc, Auth: auth})
	c.RegisterCalculatorServer(gRPCServer, &serverAPI{Calc: calc, Auth: auth})
}

func (s *serverAPI) GetTask(ctx context.Context, in *c.GetTaskRequest) (*c.GetTaskResponse, error) {
	task, err := s.Calc.GetTaskById(in.TaskId, in.UserId)
	if err != nil {
		if err == storage.ErrTaskNotFound {
			return &c.GetTaskResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
		return &c.GetTaskResponse{}, status.Error(codes.Internal, "could not be found")
	}
	return &c.GetTaskResponse{Task: task}, nil
}

func (s *serverAPI) GetDelays(ctx context.Context, in *c.Empty) (*c.GetDelaysResponse, error) {
	reult, err := s.Calc.GetDelays()
	if err != nil {
		return &c.GetDelaysResponse{}, status.Error(codes.Internal, "failed to read")
	}
	str, err := json.Marshal(reult)
	if err != nil {
		return &c.GetDelaysResponse{}, status.Error(codes.Internal, "failed to read")
	}
	return &c.GetDelaysResponse{Delays: string(str)}, nil
}

func (s *serverAPI) UpdateDelays(ctx context.Context, in *c.UpdateDelaysRequest) (*c.Empty, error) {
	type Request struct {
		Plus           int `json:"plus"`
		Minus          int `json:"minus"`
		Multiplication int `json:"multiplication"`
		Division       int `json:"division"`
	}
	var req Request
	err := json.Unmarshal([]byte(in.Delays), &req)
	if err != nil {
		return &c.Empty{}, status.Error(codes.InvalidArgument, err.Error())
	}
	m := make(map[string]int)
	m["plus"] = req.Plus
	m["minus"] = req.Minus
	m["multiplication"] = req.Multiplication
	m["division"] = req.Division
	err = s.Calc.UpdateDelays(m)
	if err != nil {
		return &c.Empty{}, status.Error(codes.Internal, "failed to update")
	}
	return &c.Empty{}, nil
}

func (s *serverAPI) GetWorkersInfo(ctx context.Context, in *c.Empty) (*c.GetWorkersInfoResponse, error) {
	result, _ := s.Calc.GetWorkersInfo()
	js, err := json.Marshal(result)
	if err != nil {
		return &c.GetWorkersInfoResponse{}, status.Error(codes.Internal, "failed to read")
	}
	return &c.GetWorkersInfoResponse{Workers: string(js)}, nil
}

func (s *serverAPI) GetAllTasks(ctx context.Context, in *c.GetAllTasksRequest) (*c.GetAllTasksResponse, error) {
	result, err := s.Calc.GetAllTasks(in.UserId)
	if err != nil {
		return &c.GetAllTasksResponse{}, status.Error(codes.Internal, "failed to read")
	}
	js, err := json.Marshal(result)
	if err != nil {
		return &c.GetAllTasksResponse{}, status.Error(codes.Internal, "failed to read")
	}
	return &c.GetAllTasksResponse{Tasks: string(js)}, nil

}

func (s *serverAPI) AddTask(ctx context.Context, in *c.AddTaskRequest) (*c.AddTaskResponse, error) {
	task, err := s.Calc.NewTask(int(in.UserId), in.Task)
	if err != nil {
		if err == storage.ErrUserNotFound {
			return nil, status.Error(codes.InvalidArgument, "user not found")
		}
		if err.Error() == "выражение недопустимо" {
			return nil, status.Error(codes.InvalidArgument, "invalid expression")
		}
		return nil, status.Error(codes.Internal, "failed to add")
	}
	go task.Start()
	result := map[string]interface{}{"id": task.Id, "expression": task.Expression, "status": "processing"}
	js, err := json.Marshal(result)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read")
	}
	return &c.AddTaskResponse{Task: string(js)}, nil
}

func (s *serverAPI) Login(ctx context.Context, in *c.LoginRequest) (*c.LoginResponse, error) {
	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	token, err := s.Auth.Login(in.GetName(), in.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid name or password")
		}

		return nil, status.Error(codes.Internal, "failed to login")
	}
	return &c.LoginResponse{Token: token}, nil
}

func (s *serverAPI) Register(ctx context.Context, in *c.RegisterRequest) (*c.RegisterResponse, error) {
	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	r := ValidatePassword(in.Password)
	if r != "" {
		return nil, status.Error(codes.InvalidArgument, r)
	}
	uid, err := s.Auth.Register(in.GetName(), in.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}
	return &c.RegisterResponse{UserId: uid}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, in *c.IsAdminRequest) (*c.IsAdminResponse, error) {
	isAdmin, err := s.Auth.IsAdmin(in.GetUserId())
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "failed to check admin status")
	}
	return &c.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func (s *serverAPI) GetUserInfo(ctx context.Context, in *c.GetUserInfoRequest) (*c.GetUserInfoResponse, error) {
	info, err := s.Auth.GetUserInfo(in.UserId)
	if err != nil {
		if err == storage.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to check")
	}
	return &c.GetUserInfoResponse{Name: info}, nil
}

func ValidatePassword(password string) string {
	if len(password) < 5 {
		return "Пароль должен содержать не менее 5 символов"
	}

	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return "Пароль должен содердать цифры"
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return "Пароль должен содержать загланые буквы"
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return "Пароль должен содержать строчные буквы"
	}

	if !regexp.MustCompile(`[!@#$%^&*]`).MatchString(password) {
		return "Пароль должен содержать специальный символ"
	}

	if !regexp.MustCompile(`^[A-Za-z0-9!@#$%^&*]+$`).MatchString(password) {
		return "Пароль должен содержать только английские буквы"
	}

	return ""
}
