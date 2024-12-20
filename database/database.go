// database/database.go
package database

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dizhechko/todo-list-server/settings"
	"github.com/jmoiron/sqlx"
)

type Task struct {
	ID      string `json:"id"      db:"id,omitempty"`
	Date    string `json:"date"    db:"date"`
	Title   string `json:"title"   db:"title"`
	Comment string `json:"comment" db:"comment"`
	Repeat  string `json:"repeat"  db:"repeat"`
}

type TasksStore struct {
	db *sqlx.DB
}

func NewTasksStore(db *sqlx.DB) TasksStore {
	return TasksStore{db: db}
}

type params struct {
	Date   string `db:"date"`
	Search string `db:"search"`
	Limit  int    `db:"limit"`
}

var infLog = log.New(os.Stdout, "todo INFO: ", log.Ldate|log.Ltime)

// ConnectDB создает подключение по пути dbPath
func ConnectDB(dbPath string) (*sqlx.DB, error) {
	// если не существует, то создаём бд
	dbStr := settings.EnvDBStr
	dbStr = strings.TrimPrefix(dbStr, ".")
	if dbStr == "" {
		appPath, err := os.Getwd()
		if err != nil {
			log.Fatalf("ConnectDB Ошибка: %v", err)
		}
		dbStr = filepath.Join(appPath, dbPath)
	}
	_, err := os.Stat(dbStr)

	// проверить существование бд и
	// sql-запрос с CREATE TABLE и CREATE INDEX
	if err != nil {
		if CreateDB(dbStr) != nil {
			return nil, err
		}
		infLog.Println("БД создана")
	} else {
		infLog.Println("БД существует")
	}

	db, err := sqlx.Connect("sqlite", dbStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// CreateDB создает бд по пути dbPath
func CreateDB(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		infLog.Printf("CreateDB Ошибка: %v", err)
		return err
	}
	defer db.Close()

	// Создание таблицы scheduler и индекса по полю date
	query := `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date CHAR(8) NOT NULL DEFAULT "",
		title VARCHAR(128) NOT NULL DEFAULT "",
		comment VARCHAR(1000) NOT NULL DEFAULT "",
		repeat VARCHAR(128) NOT NULL DEFAULT ""
	);     
	CREATE INDEX IF NOT EXISTS scheduler_date ON scheduler (date);
	`

	_, err = db.Exec(query)
	if err != nil {
		infLog.Printf("CreateDB Ошибка создания табл: %v", err)
		return err
	}

	return nil
}

// GetTasks - получение всех задач
func (s TasksStore) GetTasks(search string) ([]Task, error) {
	var args params
	query := ""
	query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT :limit"
	args = params{Limit: settings.Limit50}

	tasks := []Task{}
	task := Task{}

	rows, err := s.db.NamedQuery(query, args)
	if err != nil {
		return []Task{}, err
	}
	for rows.Next() {
		err = rows.StructScan(&task)
		if err != nil {
			return []Task{}, err
		}
		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		return []Task{}, err
	}

	return tasks, nil
}

// InsertTask - вставка задачи по id
func (s TasksStore) InsertTask(task Task) (lastInsertId int64, err error) {
	resultDB, err := s.db.NamedExec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)", &task)
	if err != nil {
		return 0, err
	}
	// Получаем ID последней записи
	lastInsertId, err = resultDB.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastInsertId, nil
}

// GetTaskByID - получение задачи по id
func (s TasksStore) GetTaskByID(id int) (Task, error) {
	task := Task{}
	err := s.db.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = errors.New("не найдена задача")
		}
		return Task{}, err
	}
	return task, nil
}

// UpdateTask - обновление задачи по id
func (s TasksStore) UpdateTask(task Task) error {
	result, err := s.db.NamedExec("UPDATE scheduler SET date = :date, title = :title, comment = :comment, repeat = :repeat WHERE id = :id", &task)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("не найдена задача")
	}
	return nil
}

// DeleteTaskByID - удаление задачи по id
func (s TasksStore) DeleteTaskByID(id int) error {
	result, err := s.db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("не найдена задача")
	}
	return nil
}
