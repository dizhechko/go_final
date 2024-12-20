package settings

import "os"

// Настройки по умолчанию
const (
	DateFormat = "20060102"
	DBPath     = "./scheduler.db" // Путь к базе данных
	Port       = ":7550"          // Порт сервера
	WebDir     = "./web"          // Директория для web файлов
)

// Лимиты на получение строк в SQL-запросах
const (
	Limit50 int = 70
)

var EnvDBStr = os.Getenv("TODO_DBFILE") // Файл БД из переменной окружения TODO_DBFILE
var EnvPortStr = os.Getenv("TODO_PORT") // Порт из переменной окружения TODO_PORT
//var EnvPass = os.Getenv("TODO_PASSWORD") // Пароль из переменной окружения TODO_PASSWORD
