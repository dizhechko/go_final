package main

import (
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/dizhechko/todo-list-server/database"
	"github.com/dizhechko/todo-list-server/handlers"
	"github.com/dizhechko/todo-list-server/settings"
	"github.com/go-chi/chi"
)

func main() {
	infLog := log.New(os.Stdout, "todo INFO: ", log.Ldate|log.Ltime)
	errLog := log.New(os.Stderr, "todo ERROR: ", log.Ldate|log.Ltime)
	infLog.Println("Старт")

	// инициализация маршрутизатора
	router := chi.NewRouter()

	// файловый сервер
	fs := http.FileServer(http.Dir(settings.WebDir))
	router.Handle("/*", http.StripPrefix("/", fs))

	// Соединение с базой данных
	db, err := database.ConnectDB(settings.DBPath)
	if err != nil {
		errLog.Println(err)
		return
	}
	defer db.Close()
	store := database.NewTasksStore(db)

	// Обработчики API
	apiRouter := chi.NewRouter()

	apiRouter.Get("/tasks", handlers.GetTasks(store))

	apiRouter.Route("/task", func(r chi.Router) {

		r.Get("/", handlers.GetTaskByID(store))

		r.Post("/", handlers.PostTask(store))

		r.Post("/done", handlers.PostTask_Done(store))

		r.Put("/", handlers.PutTask(store))

		r.Delete("/", handlers.DeleteTask(store))
	})
	router.Mount("/api", apiRouter)

	router.Get("/api/nextdate", handlers.NextDateHandler)

	port := ":" + settings.EnvPortStr
	if port == ":" {
		port = settings.Port
	}

	infLog.Printf("Стартуем на порту %s...\n", port)

	if err := http.ListenAndServe(port, router); err != nil {
		errLog.Printf("Ошибка старта: %s", err.Error())
		return
	}
}
