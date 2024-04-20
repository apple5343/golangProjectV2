package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func createTables(db *sql.DB) error {
	const op = "storage.sqlite.createTables"
	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS 
	users (
		id	INTEGER,
		name	TEXT UNIQUE,
		passwordHash	TEXT,
		isAdmin INTEGER,
		PRIMARY KEY(id AUTOINCREMENT)
	);`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS 
	tasks (
		id	INTEGER,
		expression	TEXT,
		status	TEXT,
		result	TEXT,
		created	TEXT,
		lastPing	TEXT,
		lastStep TEXT,
		userID INTEGER,
		PRIMARY KEY(id AUTOINCREMENT)
	);`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS 
	subtasks (
		id	INTEGER UNIQUE,
		value	TEXT,
		time TEXT,
		parentId INTEGER,
		result TEXT,
		PRIMARY KEY(id AUTOINCREMENT)
	);`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS 
	delays (
		operation TEXT,
		delay INTEGER
	);`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	delays := []string{"plus", "minus", "multiplication", "division"}
	for _, i := range delays {
		var delay int
		err = db.QueryRow("SELECT delay FROM delays WHERE operation = ?", i).Scan(&delay)
		if err != nil {
			stmt, err := db.Prepare("INSERT INTO delays (operation, delay) VALUES (?, ?)")
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
			_, err = stmt.Exec(i, 10)
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
	}
	return nil
}
