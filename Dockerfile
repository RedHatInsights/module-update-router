FROM registry.access.redhat.com/ubi9/go-toolset:latest as builder

USER root

WORKDIR /app
RUN mkdir -p /app/bin

COPY . .

RUN go mod download
RUN go build -o module-update-router .

FROM builder as final

WORKDIR /app

EXPOSE 8080 2112

ENTRYPOINT ["./module-update-router"]
