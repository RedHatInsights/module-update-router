#!/bin/bash

set -xev

IMAGE="quay.io/cloudservices/module-update-router"
IMAGE_TAG=$(git rev-parse --short=7 HEAD)

docker build -t "${IMAGE}:${IMAGE_TAG}" .

if [[ -n "$QUAY_USER" && -n "$QUAY_TOKEN" ]]; then
    DOCKER_CONF="$PWD/.docker"
    mkdir -p "$DOCKER_CONF"
    docker --config "$DOCKER_CONF" login --username "$QUAY_USER" --password "$QUAY_TOKEN" quay.io
    docker --config "$DOCKER_CONF" push "${IMAGE}:${IMAGE_TAG}"
fi
