APP=e2u.io/go-transit
PWD=$(shell pwd)
BUILD_DIR=$(PWD)/builds
OBJ=go-transit
REPOSITORY=go-transit

BUILD_TAG=$(date '+%Y%m%d')

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}
  

.PHONY: build
build:
	go build -o ${BUILD_DIR}/${OBJ}


.PHONY: build-linux
build-linux:
	GOARCH=amd64 GOOS=linux go build -o ${BUILD_DIR}/linux_amd64/${OBJ}
  
# 编译并构建 docker 镜像
# make build-docker REPOSITORY=
.PHONY: build-docker
build-docker: build-linux
	docker build -t ${REPOSITORY} .

.PHONY: run
run:
	go run main.go httpd.go  config.go config_file.go -f etc/config.json


.PHONY: run-docker
run-docker: build-docker
	docker-compose up --build