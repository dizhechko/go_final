FROM golang:1.22.1

WORKDIR /app

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV TODO_PORT=7540
ENV TODO_DBFILE="scheduler.db"


COPY . .

RUN go mod download

RUN go build -o ./todo_app 

EXPOSE ${TODO_PORT}

CMD ["./todo_app"]
