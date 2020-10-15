FROM quay.io/app-sre/golang:1.13

WORKDIR /go/src/app
COPY . .

RUN go get -d .
RUN go install -v .

EXPOSE 8080 2112

ENTRYPOINT ["module-update-router"]
