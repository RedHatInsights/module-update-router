FROM registry.access.redhat.com/ubi8/go-toolset as builder
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
USER root
RUN go build -o module-update-router .

FROM registry.redhat.io/ubi8/ubi-minimal
WORKDIR /
COPY --from=builder /go/src/app/module-update-router /module-update-router

USER 1001
EXPOSE 8080 2112

ENTRYPOINT ["/module-update-router"]
