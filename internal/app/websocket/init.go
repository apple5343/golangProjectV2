package websocket

func Init(secret string) *Manager {
	manager := NewManager(secret)
	return manager
}
