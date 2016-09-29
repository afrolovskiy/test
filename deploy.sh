#!/bin/bash
GOOS=linux go build -o=test-client client/main.go
GOOS=linux go build -o=test-server server/main.go

scp -i $SSH_KEY test-client ubuntu@$HOST_CLIENT:/home/ubuntu/
scp -i $SSH_KEY test-server ubuntu@$HOST_SERVER:/home/ubuntu/
 