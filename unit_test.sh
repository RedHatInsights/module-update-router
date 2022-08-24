#!/bin/bash

set -xev

mkdir -p artifacts
go install github.com/jstemmer/go-junit-report/v2@latest
go test -v -race 2>&1 | ~/go/bin/go-junit-report -set-exit-code > artifacts/junit-report.xml
