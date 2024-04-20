package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/apple5343/golangProjectV2/internal/domain/models"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrTaskNotFound = errors.New("task is not found")
)

type SqlDB struct {
	db *sql.DB
}

func OpenStorage(path string) (*SqlDB, error) {
	const op = "storage.sqlite.New"
	database, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	db := SqlDB{db: database}

	err = createTables(database)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &db, nil
}

func (s *SqlDB) AddUser(user models.User) (int64, error) {
	stmt, err := s.db.Prepare("INSERT INTO users (name, passwordHash, isAdmin) VALUES (?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(user.Name, user.PasswordHash, user.IsAdmin)
	if err != nil {
		return 0, ErrUserExists
	}
	id, _ := result.LastInsertId()
	return id, nil
}

func (s *SqlDB) CheckLogin(name, passwordHash string) (int64, error) {
	var id int64
	err := s.db.QueryRow("SELECT id FROM users WHERE name = ? and passwordHash = ?", name, passwordHash).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *SqlDB) IsAdmin(userID int) (bool, error) {
	var isAdmin int
	err := s.db.QueryRow("SELECT isAdmin FROM users WHERE id = ?", userID).Scan(&isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrUserNotFound
		}
		return false, err
	}
	if isAdmin == 0 {
		return false, nil
	}
	return true, nil
}

func (s *SqlDB) IsExist(name string) (models.User, error) {
	var user models.User
	err := s.db.QueryRow("SELECT id, name, passwordHash, isAdmin FROM users WHERE name = ?", name).Scan(&user.ID, &user.Name, &user.PasswordHash, &user.IsAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, ErrUserNotFound
		}
		return user, err
	}
	return user, nil
}

func (s *SqlDB) AddTask(task string, userID int, created time.Time) (int, error) {
	_, err := s.GetUserInfo(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	statement, err := s.db.Prepare("INSERT INTO tasks (expression, status, result, created, lastPing, lastStep, userID) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer statement.Close()
	res, err := statement.Exec(task, "processing", "", created.Format("2006-01-02 15:04:05"), "", task, userID)
	if err != nil {
		return 0, err
	}
	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(lastInsertId), nil
}

func (s *SqlDB) SetResult(taskId int, result string) error {
	statement, err := s.db.Prepare("UPDATE tasks SET result = ? WHERE id = ?")
	defer statement.Close()
	_, err = statement.Exec(result, taskId)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlDB) SetStatus(taskId int, newStatus string) error {
	statement, err := s.db.Prepare("UPDATE tasks SET status = ? WHERE id = ?")
	defer statement.Close()
	_, err = statement.Exec(newStatus, taskId)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlDB) GetAllTasks(userID int64) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	rows, err := s.db.Query("SELECT * FROM tasks WHERE userID = ?", userID)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return results, err
	}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			log.Fatal(err)
		}
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := pointers[i].(*interface{})
			rowMap[colName] = *val
		}

		results = append(results, rowMap)
	}
	if err := rows.Err(); err != nil {
		return results, err
	}
	return results, nil
}

func (s *SqlDB) GetDelays() (map[string]int, error) {
	result := make(map[string]int)
	rows, err := s.db.Query("SELECT * FROM delays")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	var op string
	var delay int
	for rows.Next() {
		err := rows.Scan(&op, &delay)
		if err != nil {
			return result, err
		}
		result[op] = delay
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func (s *SqlDB) UpdateDelays(newDelays map[string]int) error {
	for k, v := range newDelays {
		statement, err := s.db.Prepare("UPDATE delays SET delay = ? WHERE operation = ?")
		defer statement.Close()
		_, err = statement.Exec(v, k)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SqlDB) AddSubtask(value string, tim time.Time, parentId int, updated string) error {
	statement, err := s.db.Prepare("INSERT INTO subtasks (value, time, parentId, result) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(value, tim.Format("2006-01-02 15:04:05"), parentId, updated)
	if err != nil {
		return err
	}
	return err
}

func (s *SqlDB) GetSubtasks(parentId int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	rows, err := s.db.Query("SELECT * FROM subtasks WHERE parentId = ?", parentId)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return results, err
	}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			log.Fatal(err)
		}
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := pointers[i].(*interface{})
			rowMap[colName] = *val
		}

		results = append(results, rowMap)
	}
	if err := rows.Err(); err != nil {
		return results, err
	}
	return results, nil
}

func (s *SqlDB) GetTaskById(taskId, userID int64) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	var id, c int
	var expression, status, result, created, lastPing, lastStep string
	row := s.db.QueryRow("SELECT * FROM tasks WHERE id = ? AND userID = ?", taskId, userID)
	err := row.Scan(&id, &expression, &status, &result, &created, &lastPing, &lastStep, &c)
	if err != nil {
		if err == sql.ErrNoRows {
			return res, ErrTaskNotFound
		}
		return res, err
	}
	res["id"] = id
	res["expression"] = expression
	res["status"] = status
	res["result"] = result
	res["created"] = created
	res["lastPing"] = lastPing
	res["lastStep"] = lastStep
	subtasks, err := s.GetSubtasks(int(taskId))
	if err != nil {
		return res, err
	}
	res["subtasks"] = subtasks
	return res, nil
}

func (s *SqlDB) UpdatePing(taskId int, tim time.Time) error {
	statement, err := s.db.Prepare("UPDATE tasks SET lastPing = ? WHERE id = ?")
	defer statement.Close()
	_, err = statement.Exec(tim.Format("2006-01-02 15:04:05"), taskId)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlDB) GetInterruptedTasks() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	rows, err := s.db.Query("SELECT * FROM tasks WHERE status = ?", "processing")
	if err != nil {
		return results, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return results, err
	}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			log.Fatal(err)
		}
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := pointers[i].(*interface{})
			rowMap[colName] = *val
		}

		results = append(results, rowMap)
	}
	if err := rows.Err(); err != nil {
		return results, err
	}
	return results, nil
}

func (s *SqlDB) UpdateLastStep(id int, step string) error {
	statement, err := s.db.Prepare("UPDATE tasks SET lastStep = ? WHERE id = ?")
	defer statement.Close()
	_, err = statement.Exec(step, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlDB) GetUserInfo(userID int) (string, error) {
	var result string
	err := s.db.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, ErrUserNotFound
		}
		return result, err
	}
	return result, nil
}
