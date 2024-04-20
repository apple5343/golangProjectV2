package app

import "net/http"

func (s *Server) SetupRoutes() {
	fs := http.FileServer(http.Dir("internal/static"))
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	s.router.HandleFunc("/auth", s.Auth())
	s.router.Handle("/login", s.Login())
	s.router.Handle("/register", s.Register())
	s.router.Handle("/getInfo", s.GetUserInfo())
	s.router.Handle("/addTask", s.AddTask())
	s.router.Handle("/getTasks", s.GetTasks())
	s.router.Handle("/getTask", s.GetTask())
	s.router.Handle("/getWorkersInfo", s.GetWorkersInfo())
	s.router.Handle("/logout", s.Logout())
	s.router.Handle("/updateDelays", s.UpdateDelays())
	s.router.Handle("/getDelays", s.GetDelays())
	s.router.Handle("/ws", s.manager.ServeWs(store))
	s.router.HandleFunc("/", s.Home())
}
