FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder

USER root
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o module-update-router .

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS final

COPY --from=builder --chmod=755 /app/module-update-router /usr/local/bin/module-update-router
USER 1001

EXPOSE 8080 2112
ENTRYPOINT ["/usr/local/bin/module-update-router"]
