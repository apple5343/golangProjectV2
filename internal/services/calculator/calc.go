package calculator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/apple5343/golangProjectV2/internal/app/websocket"
	"github.com/apple5343/golangProjectV2/internal/config"
	storage "github.com/apple5343/golangProjectV2/internal/storage/sqlite"
	"golang.org/x/exp/slices"
)

type Calculator struct {
	toProcess   chan *Subtask
	db          storage.SqlDB
	Worker      Worker
	Updates     chan websocket.Event
	UpdatesTask chan TaskUpdate
}

type Worker struct {
	mu        sync.Mutex
	list      []map[string]interface{}
	Delays    map[string]int
	toProcess chan *Subtask
	Updates   chan websocket.Event
}

type Task struct {
	db         storage.SqlDB
	subtask    *Spliter
	toProcess  chan *Subtask
	Id         int
	Status     string
	Expression string
	Result     string
	Created    time.Time
	LastPing   time.Time
	UserID     int
	UpdateCh   chan TaskUpdate
}

type Subtask struct {
	substack *Symbol
	pingCh   chan int
	wg       *sync.WaitGroup
	taskId   int
}

type TaskUpdate struct {
	UserId   int
	TaskId   int
	LastPing string
	IsDone   int
	Result   string
}

func (c *Calculator) listenTasksUpdate(ch <-chan TaskUpdate) {
	for {
		select {
		case task, ok := <-ch:
			if !ok {
				return
			}
			update := websocket.UpdateTaskMessage(task.UserId, task.TaskId, task.IsDone, task.LastPing, task.Result)
			c.Updates <- *update
		}
	}
}

func NewCalculator(cfg *config.Config, db storage.SqlDB, ch chan websocket.Event) (*Calculator, error) {
	toProcess := make(chan *Subtask)
	list := []map[string]interface{}{}
	calculator := &Calculator{toProcess: toProcess, db: db, Updates: ch}
	for i := 0; i < 4; i++ {
		killCh := make(chan int)
		list = append(list, map[string]interface{}{"id": i + 1, "kill": killCh, "status": "in waiting", "expression": "", "expressionId": 0})
		go calculator.Worker.worker(toProcess, i+1, killCh)
	}
	tasksCh := make(chan TaskUpdate)
	calculator.UpdatesTask = tasksCh
	go calculator.listenTasksUpdate(tasksCh)
	delays, err := db.GetDelays()
	if err != nil {
		return calculator, err
	}
	calculator.Worker.Updates = ch
	calculator.Worker.Delays = delays
	calculator.Worker.list = list
	calculator.Worker.toProcess = toProcess
	return calculator, nil
}

func (w *Worker) ChangeStatus(id int, newStatus string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, k := range w.list {
		if k["id"] == id {
			k["status"] = newStatus
			return
		}
	}
}

func (w *Worker) ChangeExpression(id int, expression string, expressionId int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, k := range w.list {
		if k["id"] == id {
			k["expression"] = expression
			k["expressionId"] = expressionId
			return
		}
	}
}

func (w *Worker) GetWorkersInfo() []map[string]interface{} {
	result := []map[string]interface{}{}
	for _, v := range w.list {
		result = append(result, map[string]interface{}{"id": v["id"], "status": v["status"], "expression": v["expression"], "expressionId": v["expressionId"]})
	}
	return result
}

func (w *Worker) RemoveWorker(id int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	workers := []map[string]interface{}{}
	for _, v := range w.list {
		if v["id"] != id {
			workers = append(workers, v)
		} else {
			v["kill"].(chan int) <- 0
		}
	}
	w.list = workers
	if len(w.list)-1 == 1 {
		i := w.list[len(w.list)-1][`id`].(int) + 1
		killCh := make(chan int)
		w.list = append(w.list, map[string]interface{}{"id": i, "kill": killCh, "status": "in waiting", "expression": "", "expressionId": 0})
		go w.worker(w.toProcess, i, killCh)
	}
}

func (w *Worker) worker(in chan *Subtask, id int, killCh <-chan int) {
	for {
		select {
		case v := <-in:
			w.ChangeStatus(id, "at work")
			expression, _ := govaluate.NewEvaluableExpression(v.substack.value)
			w.ChangeExpression(id, v.substack.value, v.taskId)
			update := websocket.UpdateWorkerMessage(id, "at work", v.substack.value, v.taskId)
			w.Updates <- *update
			select {
			case <-time.After(time.Duration(w.Delays[v.substack.op]) * time.Second):
				result, _ := expression.Eval(nil)
				v.substack.result = strconv.FormatFloat(result.(float64), 'f', -1, 64)
				v.pingCh <- v.substack.id
				v.wg.Done()
				w.ChangeStatus(id, "in waiting")
				update := websocket.UpdateWorkerMessage(id, "in waiting", "", 0)
				w.Updates <- *update
				w.ChangeExpression(id, "", 0)
			case <-killCh:
				in <- v
				return
			}
		case <-killCh:
			return
		}
	}
}

func (c *Calculator) NewTask(userID int, expression string) (*Task, error) {
	expression = strings.ReplaceAll(expression, " ", "")
	if !IsValidExpression(expression) {
		return nil, fmt.Errorf("выражение недопустимо")
	}
	t := time.Now()
	id, err := c.db.AddTask(expression, userID, t)
	if err != nil {
		return nil, err
	}
	spliter := NewSpliter(expression)
	task := &Task{subtask: spliter, toProcess: c.toProcess, Expression: expression, db: c.db, Created: t, UserID: userID, Id: id, UpdateCh: c.UpdatesTask}
	return task, nil
}

func (t *Task) Start() {
	wg := &sync.WaitGroup{}
	for !t.subtask.done {
		t.subtask.Split()
		resultsCh := make(chan int)
		for _, v := range t.subtask.Symbols {
			if v.expressionType == "calculation" {
				go func(s *Symbol) {
					t.toProcess <- &Subtask{substack: s, pingCh: resultsCh, wg: wg, taskId: t.Id}
				}(v)
				wg.Add(1)
			}
		}
		go func() {
			wg.Wait()
			close(resultsCh)
		}()
		processed := []int{}
		for i := range resultsCh {
			processed = append(processed, i)
			updated := ""
			old := ""
			lastStep := ""
			for _, v := range t.subtask.Symbols {
				if v.id == i {
					i = 99999999999
					old += `<span class=old>` + v.value + `</span>`
					updated += `<span class=new>` + v.result + `</span>`
					lastStep += v.result
					continue
				}
				if v.result != "" && slices.IndexFunc(processed, func(i int) bool { return i == v.id }) >= 0 {
					updated += v.result
					old += v.result
					lastStep += v.result
				} else {
					lastStep += v.value
					updated += v.value
					old += v.value
				}
			}
			tm := time.Now()
			err := t.db.UpdatePing(t.Id, tm)
			if err != nil {
				fmt.Println(err)
			}
			err = t.db.AddSubtask(old, tm, t.Id, updated)
			if err != nil {
				fmt.Println(err)
			}
			err = t.db.UpdateLastStep(t.Id, lastStep)
			if err != nil {
				fmt.Println(err)
			}
			t.UpdateCh <- TaskUpdate{t.UserID, t.Id, tm.Format("2006-01-02 15:04:05"), 0, t.Result}
		}
		newExpression := ""
		for _, v := range t.subtask.Symbols {
			if v.result != "" {
				newExpression += v.result
			} else {
				newExpression += v.value
			}
		}
		t.subtask.Update(newExpression)
	}
	t.Result = t.subtask.result
	t.UpdateCh <- TaskUpdate{t.UserID, t.Id, time.Now().Format("2006-01-02 15:04:05"), 1, t.Result}
	t.db.SetResult(t.Id, t.Result)
	t.db.SetStatus(t.Id, "completed")
}

func (c *Calculator) ContinueCalculations() {
	tasks, err := c.db.GetInterruptedTasks()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range tasks {
		lastStep, _ := c.db.GetTaskById(v["id"].(int64), v["userID"].(int64))
		spliter := NewSpliter(lastStep["lastStep"].(string))
		t, err := time.Parse("2006-01-02 15:04:05", v["created"].(string))
		if err != nil {
			fmt.Println(err)
		}
		task := &Task{subtask: spliter, toProcess: c.toProcess, Expression: lastStep["lastStep"].(string), db: c.db, Created: t, Id: int(v["id"].(int64)),
			UserID: int(v["userID"].(int64)), UpdateCh: c.UpdatesTask}
		go task.Start()
	}
}

func (c *Calculator) GetAllTasks(userID int64) ([]map[string]interface{}, error) {
	res, err := c.db.GetAllTasks(userID)
	return res, err
}

func (c *Calculator) GetDelays() (map[string]int, error) {
	res, err := c.db.GetDelays()
	return res, err
}

func (c *Calculator) GetTaskById(taskID, userID int64) (string, error) {
	res, err := c.db.GetTaskById(taskID, userID)
	if err != nil {
		return "", err
	}
	js, err := json.Marshal(res)
	return string(js), err
}

func (c *Calculator) GetWorkersInfo() ([]map[string]interface{}, error) {
	return c.Worker.GetWorkersInfo(), nil
}

func (c *Calculator) UpdateDelays(delays map[string]int) error {
	err := c.db.UpdateDelays(delays)
	if err == nil {
		for k, v := range delays {
			c.Worker.Delays[k] = v
		}
	}
	return err
}
