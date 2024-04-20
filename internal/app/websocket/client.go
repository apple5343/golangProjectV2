package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool
type OnlineCliets map[int]*Client

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	egress     chan Event
	id         int
	isAdmin    bool
}

func NewClient(conn *websocket.Conn, manager *Manager, id int, isAdmin bool) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
		id:         id,
		isAdmin:    isAdmin,
	}
}

func (c *Client) WriteMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()
	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println(err)
				}
				return
			}
			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println(err)
			}
		}
	}
}
