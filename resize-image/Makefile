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
	oci os object put -bn environments --file .env.yaml --name cloud-functions/${BINARY_NAME}/.env.yaml --no-multipart --force

env-down:
	oci os object get -bn environments --file .env --name cloud-functions/${BINARY_NAME}/.env
	oci os object get -bn environments --file .env.yaml --name cloud-functions/${BINARY_NAME}/.env.yaml

run:
	go run cmd/main.go

run-docker:
	gcloud alpha functions local deploy gcp-resize-image \
		--entry-point=function.go \
		--runtime=go122 \
		--env-vars-file=.env.yaml

call:
	gcloud alpha functions local call gcp-resize-image --cloud-event='{ \
		 "specversion" : "1.0",\
		  "type" : "com.github.pull.create", \
		  "source" : "https://github.com/cloudevents/spec/pull", \
		  "subject" : "123", \
		  "id" : "ce", \
		  "time" : "2021-01-27T18:30:00Z", \
		  "data" : {"subscription":"projects/sayho-general/subscriptions/resize-image","message":{"data":{"objectName":"mongoose.jpg","targetName":"ae30f908-8723-4216-ba6b-c798772d0773.jpg","targetWidth":0,"targetheight":400}}} \
		}'\

delete:
	gcloud alpha functions local delete gcp-resize-image

deploy:
	gcloud functions deploy resize-image \
		--memory=2GiB \
		--gen2 \
		--region=asia-northeast3 \
		--runtime=go122 \
		--source=/Users/sayho/GolandProjects/cloud-functions/${BINARY_NAME} \
		--entry-point=function.go \
		--trigger-topic=resize-image \
		--env-vars-file=.env.yaml
