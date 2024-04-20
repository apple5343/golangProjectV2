package websocket

import (
	"encoding/json"
	"log"
)

type Event struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	To      int    `json:"to"`
}

type EventHandler func(event Event, c *Client) error

const (
	EventTaskUpdate   = "update task"
	EventWorkerUpdate = "update worker"
)

func UpdateTaskMessage(userId, taskId, isDone int, lastPing, result string) *Event {
	type UpdateTask struct {
		UserId   int    `json:"userId"`
		TaskId   int    `json:"taskId"`
		LastPing string `json:"lastPing"`
		IsDone   int    `json:"isDone"`
		Result   string `json:"result"`
	}
	task := UpdateTask{
		TaskId:   taskId,
		LastPing: lastPing,
		IsDone:   isDone,
		Result:   result,
		UserId:   userId,
	}
	message, err := json.Marshal(task)
	if err != nil {
		log.Println(err)
	}
	return &Event{
		Type:    EventTaskUpdate,
		Message: string(message),
		To:      task.UserId,
	}
}

func UpdateWorkerMessage(workerId int, state, exp string, taskId int) *Event {
	type UpdateWorker struct {
		Id     int    `json:"id"`
		State  string `json:"state"`
		Exp    string `json:"exp"`
		TaskId int    `json:"taskId"`
	}
	update := UpdateWorker{
		Id:     workerId,
		State:  state,
		Exp:    exp,
		TaskId: taskId,
	}
	message, err := json.Marshal(update)
	if err != nil {
		log.Println(err)
	}
	return &Event{
		Type:    EventWorkerUpdate,
		Message: string(message),
		To:      0,
	}
}
