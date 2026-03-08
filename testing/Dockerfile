FROM golang:1.26

WORKDIR /usr/src/app

COPY go.mod main.go ./

RUN go build -v -o /usr/local/bin/app .

CMD ["app"]