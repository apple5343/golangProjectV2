package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/apple5343/golangProjectV2/internal/lib/jwt"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients         ClientList
	online          OnlineCliets
	mu              sync.RWMutex
	secret          string
	ListenUpdatesCh chan Event
}

func NewManager(secret string) *Manager {
	ch := make(chan Event)
	m := &Manager{
		clients:         make(ClientList),
		online:          make(OnlineCliets),
		secret:          secret,
		ListenUpdatesCh: ch,
	}
	go m.listenUpdates()
	return m
}

func (m *Manager) listenUpdates() {
	for {
		select {
		case event, ok := <-m.ListenUpdatesCh:
			if !ok {
				return
			}
			if event.Type == EventTaskUpdate {
				client, ok := m.online[event.To]
				if ok {
					client.egress <- event
				}
			} else if event.Type == EventWorkerUpdate {
				for k := range m.online {
					client := m.online[k]
					if client.isAdmin {
						client.egress <- event
					}
				}
			}
		}
	}
}

func SendMessage(event Event, c *Client) error {
	c.egress <- event
	return nil
}

func (m *Manager) ServeWs(store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		session, _ := store.Get(r, "session")
		token, ok := session.Values["token"]
		if !ok {
			http.Error(w, "No token", http.StatusUnauthorized)
			return
		}
		claims, err := jwt.TokenValues(token.(string), m.secret)
		if err != nil {
			http.Error(w, "Неправильный токен", http.StatusUnauthorized)
			return
		}
		idValue, ok := claims["id"]
		if !ok {
			http.Error(w, "Неправильный токен", http.StatusUnauthorized)
			return
		}
		id, ok := idValue.(float64)
		if !ok {
			http.Error(w, "ID format error", http.StatusUnauthorized)
			return
		}
		isAdmin, ok := claims["isAdmin"]
		if !ok {
			client := NewClient(conn, m, int(id), false)
			m.addClient(client)
			go client.WriteMessages()
			return
		}
		if isAdmin.(float64) == 1 {
			client := NewClient(conn, m, int(id), true)
			m.addClient(client)
			go client.WriteMessages()
			return
		} else {
			client := NewClient(conn, m, int(id), false)
			m.addClient(client)
			go client.WriteMessages()
		}
	}
}

func (m *Manager) addClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client] = true
	m.online[client.id] = client
}

func (m *Manager) removeClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.online, client.id)
		delete(m.clients, client)
	}
}
