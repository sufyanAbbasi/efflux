all: build run

gen:
	grpc_tools_node_protoc -I=./ --go_out=go/ --go_opt=paths=source_relative --js_out=public/ efflux.proto

go_build:
	(cd go; go build .)

go_run:
	./go/efflux

golang: gen go_build go_run

rust_build:
	(cd rust; cargo build)

rust_run:
	./rust/target/debug/efflux

rs: gen rust_build rust_run

build: gen go_build rust_build

run: go_run