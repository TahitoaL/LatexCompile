FROM golang:1.18.3-bullseye

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./...

RUN apt update && apt install -y texlive-full
RUN go build 

CMD ["app"]

EXPOSE 3000
