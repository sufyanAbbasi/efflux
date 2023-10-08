all: run

gen:
	grpc_tools_node_protoc -I=./ --go_out=go/ --go_opt=paths=source_relative --js_out=public/ efflux.proto

go_build:
	(cd go; go build -o ../ .)

build: gen go_build

run: build
	./efflux