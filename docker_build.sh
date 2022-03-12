#!/bin/bash
APP="weisuo"
docker run --rm -v "$PWD":/usr/src/$APP -w /usr/src/$APP golang:1.17 go build -v
