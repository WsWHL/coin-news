FROM golang:latest AS build-stage

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct && \
    go mod download

COPY ./src ./src

RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/src/app/news ./src/main.go

FROM chromedp/headless-shell:latest AS release-stage

WORKDIR /app

COPY --from=build-stage /usr/src/app/news /app/news

RUN apt-get update && \
    apt-get install -y chromium && \
    mkdir -p /app/logs/

EXPOSE 8080

ENTRYPOINT ["/app/news"]