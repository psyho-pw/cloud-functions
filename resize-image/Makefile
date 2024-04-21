BINARY_NAME=resize-image

# .env 파일 존재 여부 확인
ifeq ($(wildcard .env),)
    $(error .env file not found)
endif

# .env 파일 로드
include .env
export

env-up:
	oci os object put -bn environments --file .env --name cloud-functions/${BINARY_NAME}/.env --no-multipart --force

env-down:
	oci os object get -bn environments --file .env --name cloud-functions${BINARY_NAME}/.env

run:
	go run cmd/main.go

all: run
