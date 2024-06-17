FROM golang:1.22.1-bullseye

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /agent

EXPOSE 8090

ENTRYPOINT ["/agent"]