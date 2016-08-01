APP=e2u.io/go-transit
PWD=$(shell pwd)
BUILD_DIR=$(PWD)/builds
OBJ=go-transit
REPOSITORY=go-transit

BUILD_TAG=-ldflags="-X main.Version=$(shell date '+%Y%m%d%H%M%S')"

# -ldflags="-X main.who=Space -X main.ENV=PROD"

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

.PHONY: reformat
reformat:
	vet -all .
	goreturns -w -d -e -l .
	goimports -w -d -e -l .
	gotype -a -v .

.PHONY: build
build: reformat
	go build -o ${BUILD_DIR}/${OBJ} ${BUILD_TAG}


.PHONY: build-linux
build-linux: reformat
	GOARCH=amd64 GOOS=linux go build -o ${BUILD_DIR}/linux_amd64/${OBJ} ${BUILD_TAG}
  
# 编译并构建 docker 镜像
# make build-docker REPOSITORY=
.PHONY: build-docker
build-docker: build-linux
	docker build -t ${REPOSITORY} .

.PHONY: run
run: reformat
	go run access_user.go config.go config_file.go httpd.go  main.go -f ./etc/config.json -u ./etc/access_users.json


.PHONY: run-docker
run-docker: build-docker
	docker-compose up --build
