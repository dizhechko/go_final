// handlers/handlers.go
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dizhechko/todo-list-server/database"
	"github.com/dizhechko/todo-list-server/scheduler"
	"github.com/dizhechko/todo-list-server/settings"
)

/*
	type Claims struct {
		//Exp      int64  `json:"exp"`
		Checksum string `json:"checksum"`
		jwt.StandardClaims
	}
*/
type TaskID struct {
	ID int64 `json:"id"`
}

// errorJSON возвращает json-строку с ошибкой
func errorJSON(err error) string {
	jsonError, err := json.Marshal(map[string]string{"error": err.Error()})
	if err != nil {
		println(err)
		return ""
	}
	return string(jsonError)
}

// NextDateHandler получает следующую дату повторения задачи по параметрам now, date, repeat
func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	//Получаем параметры из запроса
	now := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	//Вычисляем следующую дату
	nextDate, err := scheduler.NextDate(now, date, repeat)
	if err != nil {
		log.Printf("NextDateHandler: %v %v %v %v; error: %v\n", now, date, repeat, nextDate, err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(""))
		return
	}

	//Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(nextDate))
}

// GetTasks возвращает все задачи из БД в формате списка JSON
func GetTasks(store database.TasksStore) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			err := errors.New("метод не поддерживается")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		search := r.URL.Query().Get("search")
		tasks, err := store.GetTasks(search)
		if err != nil {
			log.Printf("Handlers.GetTasks: search = %v; err = %v\n", search, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
		}

		response := map[string][]database.Task{"tasks": tasks}
		jsonResponse, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonResponse)
	}
}

// GetTaskByID возвращает задачу по переданному ID
func GetTaskByID(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idParam := r.URL.Query().Get("id")
		if strings.TrimSpace(idParam) == "" {
			err := errors.New("Не найден ID")
			log.Printf("Handlers.GetTaskByID: id = %v; error = %v\n", idParam, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		task, err := store.GetTaskByID(id)
		if err != nil {
			log.Printf("Handlers.GetTaskByID: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		resp, err := json.MarshalIndent(&task, "", "  ")
		if err != nil {
			log.Printf("Handlers.GetTaskByID: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}
}

// PostTask создает новую задачу по переданным в http-запросе параметрам и записывает в БД
func PostTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			err := errors.New("метод не поддерживается")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		task := database.Task{}
		var buf bytes.Buffer

		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			log.Printf("Handlers.PostTask: buf = %v; error = %v\n", buf, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
			log.Printf("Handlers.PostTask: task = %v; error = %v\n", task, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		// проверяем корректность title, date, repeat, и корректируем
		if task.Title == "" {
			err := errors.New("заголовок не определен")
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			log.Println(err)
			return
		}

		date := strings.TrimSpace(task.Date)
		now := time.Now().Format(settings.DateFormat)
		nextDate := ""

		if len(date) == 0 {
			task.Date = now
		} else {
			begDate, err := time.Parse(settings.DateFormat, date)
			if err != nil {
				http.Error(w, errorJSON(err), http.StatusBadRequest)
				return
			}
			if begDate.Before(time.Now()) {
				if repeat := strings.TrimSpace(task.Repeat); repeat == "" {
					task.Date = now
				} else {
					nextDate, err = scheduler.NextDate(now, date, task.Repeat)
					if err != nil {
						http.Error(w, errorJSON(err), http.StatusBadRequest)
						return
					}
					task.Date = nextDate
				}
			}
		}

		lastID, err := store.InsertTask(task)
		if err != nil {
			log.Printf("Handlers.PostTask: task = %v; error = %v\n", task, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Формируем JSON, в формате {"id":"xxx"} и отправляем ответ
		taskID := TaskID{ID: lastID}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(taskID)
	}
}

// PostTask_Done удаляет задачу по ID когда не задано правило повторения
// или обновляет дату следующим повторением по правилу указанному в задаче
func PostTask_Done(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			err := errors.New("метод не поддерживается")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		idParam := r.URL.Query().Get("id")

		if strings.TrimSpace(idParam) == "" {
			err := errors.New("не определен ID")
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		// получаем задачу из БД по ID
		task, err := store.GetTaskByID(id)
		if err != nil {
			log.Printf("Handlers.PostTask_Done: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		if strings.TrimSpace(task.Repeat) == "" {

			if err := store.DeleteTaskByID(id); err != nil {
				log.Printf("Handlers.PostTask_Done: id = %v; task = %v; error = %v\n", id, task, err)
				http.Error(w, errorJSON(err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		// получаем новую дату повторения задачи и записываем в базу
		now := time.Now().Add(time.Hour * 25).Format(settings.DateFormat)
		task.Date, err = scheduler.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return

		}

		err = store.UpdateTask(task)
		if err != nil {
			log.Printf("Handlers.PostTask_Done: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
}

// PutTask обновляет задачу, переданными в json данными, по ID
func PutTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			err := errors.New("нет поддержки метода")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		task := database.Task{}
		var buf bytes.Buffer

		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
			log.Printf("Handlers.PutTask: task = %v; error = %v\n", task, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		idTask := strings.TrimSpace(task.ID)
		if idTask == "" {
			err := errors.New("не определен id задачи")
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		// проверяем корректность title, date дополнительно
		if task.Title == "" || task.Date == "" {
			err := errors.New("не определен заголовок задачи")
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			log.Println(err)
			return
		}

		date := strings.TrimSpace(task.Date)
		now := time.Now().Format(settings.DateFormat)
		nextDate := ""
		if repeat := strings.TrimSpace(task.Repeat); repeat != "" {
			nextDate, err = scheduler.NextDate(now, date, task.Repeat)
			if err != nil {
				http.Error(w, errorJSON(err), http.StatusBadRequest)
				return
			}
		}

		if len(date) == 0 {
			task.Date = now
		} else {
			begDate, err := time.Parse(settings.DateFormat, date)
			if err != nil {
				http.Error(w, errorJSON(err), http.StatusBadRequest)
				return
			}
			if begDate.Before(time.Now()) {
				if repeat := strings.TrimSpace(task.Repeat); repeat == "" {
					task.Date = now
				} else {
					task.Date = nextDate
				}
			}
		}

		err = store.UpdateTask(task)
		if err != nil {
			log.Printf("Handlers.PutTask: task = %v; error = %v\n", task, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
}

// DeleteTask удаляет задачу из базы по ID
func DeleteTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			err := errors.New("метод не определен")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		idParam := r.URL.Query().Get("id")
		if strings.TrimSpace(idParam) == "" {
			err := errors.New("ID не найден")
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		if err := store.DeleteTaskByID(id); err != nil {
			log.Printf("Handlers.DeleteTask: id = %v\n, error = %v\n", id, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
}
