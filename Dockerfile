# Build
FROM golang:1.23

WORKDIR /app

ENV APP_NAME="todo_server"
ENV TODO_PORT=7550
ENV TODO_DBFILE=../scheduler.db


COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./todo_server main.go

EXPOSE $TODO_PORT

CMD ["./todo_server"]

