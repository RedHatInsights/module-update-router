FROM golang:1.13

WORKDIR /go/src/app
COPY . .

RUN go get -d .
RUN go install -v .

CMD ["module-update-router"]
