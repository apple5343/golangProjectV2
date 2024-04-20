package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/apple5343/golangProjectV2/internal/lib/jwt"
	c "github.com/apple5343/grpc"
	"github.com/gorilla/sessions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var tpl, _ = template.ParseGlob("internal/static/templates/*.html")
var store = sessions.NewCookieStore([]byte("password"))

func GetToken(r *http.Request, secret string) (int, error) {
	session, _ := store.Get(r, "session")
	token, ok := session.Values["token"]
	if !ok {
		return 0, fmt.Errorf("No token")
	}
	claims, err := jwt.TokenValues(token.(string), secret)
	if err != nil {
		return 0, fmt.Errorf("Неправильный токен")
	}
	idValue, ok := claims["id"]
	if !ok {
		return 0, fmt.Errorf("Неправильный токен")
	}
	id, ok := idValue.(float64)
	if !ok {
		return 0, fmt.Errorf("ID format error")
	}
	return int(id), nil
}

func IsAdmin(r *http.Request, secret string) (bool, error) {
	session, _ := store.Get(r, "session")
	token, ok := session.Values["token"]
	if !ok {
		return false, fmt.Errorf("No token")
	}
	claims, err := jwt.TokenValues(token.(string), secret)
	if err != nil {
		return false, fmt.Errorf("Неправильный токен")
	}
	isAdmin, ok := claims["isAdmin"]
	if !ok {
		return false, fmt.Errorf("Неправильный токен")
	}
	return isAdmin.(float64) == 1, nil
}

func (s *Server) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		type Request struct {
			Name      string `json:"name"`
			Password  string `json:"password"`
			Password2 string `json:"password2"`
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Password != req.Password2 {
			http.Error(w, "Пароли должны совпадать", http.StatusBadRequest)
			return
		}
		_, err := s.auth.Register(context.TODO(), &c.RegisterRequest{Name: req.Name, Password: req.Password})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		w.Write([]byte("Пользователь создан"))
	}
}

func (s *Server) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		type Request struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := s.auth.Login(context.TODO(), &c.LoginRequest{Name: req.Name, Password: req.Password})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		session, _ := store.Get(r, "session")
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		session.Values["token"] = res.Token
		session.Save(r, w)
		w.Write([]byte("OK"))
	}
}

func (s *Server) AddTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		type Request struct {
			Task string `json:"task"`
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, err := GetToken(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		task, err := s.calculator.AddTask(context.TODO(), &c.AddTaskRequest{UserId: int64(id), Task: req.Task})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusUnauthorized)
				return
			}
		}
		w.Write([]byte(task.Task))
	}
}

func (s *Server) GetUserInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		id, err := GetToken(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		info, err := s.auth.GetUserInfo(context.TODO(), &c.GetUserInfoRequest{UserId: int64(id)})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		w.Write([]byte(info.Name))
	}
}

func (s *Server) Auth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tpl.ExecuteTemplate(w, "auth.html", nil)
	}
}

func (s *Server) Home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tpl.ExecuteTemplate(w, "index.html", nil)
	}
}

func (s *Server) GetTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		id, err := GetToken(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		result, err := s.calculator.GetAllTasks(context.TODO(), &c.GetAllTasksRequest{UserId: int64(id)})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(result.Tasks))
	}
}

func (s *Server) GetDelays() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		isAdmin, err := IsAdmin(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		if !isAdmin {
			http.Error(w, "Недостаточно прав", http.StatusForbidden)
		}
		result, err := s.calculator.GetDelays(context.TODO(), &c.Empty{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(result.Delays))
	}
}

func (s *Server) UpdateDelays() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		isAdmin, err := IsAdmin(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		if !isAdmin {
			http.Error(w, "Недостаточно прав", http.StatusForbidden)
		}
		type Request struct {
			Delays map[string]int `json:"delays"`
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m := make(map[string]int)
		for k, v := range req.Delays {
			if v < 0 {
				http.Error(w, "Отрицательная задержка", http.StatusBadRequest)
				return
			}
			m[k] = v
		}

		str, err := json.Marshal(m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		_, err = s.calculator.UpdateDelays(context.TODO(), &c.UpdateDelaysRequest{Delays: string(str)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) GetTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		id, err := GetToken(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		queryParams := r.URL.Query()
		taskIdStr := queryParams.Get("id")
		if taskIdStr == "" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		taskId, err := strconv.Atoi(taskIdStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		task, err := s.calculator.GetTask(context.TODO(), &c.GetTaskRequest{TaskId: int64(taskId), UserId: int64(id)})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(task.Task))
	}
}

func (s *Server) GetWorkersInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		id, err := GetToken(r, s.config.SecretJWT)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		isAdmin, err := s.auth.IsAdmin(context.TODO(), &c.IsAdminRequest{UserId: int64(id)})
		if err != nil {
			s, _ := status.FromError(err)
			if s.Code() == codes.Internal {
				http.Error(w, s.Message(), http.StatusInternalServerError)
				return
			} else {
				http.Error(w, s.Message(), http.StatusBadRequest)
				return
			}
		}
		if !isAdmin.IsAdmin {
			http.Error(w, "Недостаточно прав", http.StatusForbidden)
		}
		result, err := s.calculator.GetWorkersInfo(context.TODO(), &c.Empty{})
		if err != nil {
			s, _ := status.FromError(err)
			http.Error(w, s.Message(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(result.Workers))
	}
}

func (s *Server) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}
		delete(session.Values, "token")
		session.Save(r, w)
	}
}
