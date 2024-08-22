all: install build run

install:
	npm install grpc-tools --global
	command -v protoc-gen-go >/dev/null 2>&1 || go install github.com/golang/protobuf/protoc-gen-go@latest
	command -v cargo >/dev/null 2>&1 || curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

install:
	npm -i
	# See https://github.com/tikv/grpc-rs
	cargo install protobuf-codegen
	cargo install grpcio-compiler

gen:
	grpc_tools_node_protoc -I=./ --rust_out=rust/src/efflux/ --go_out=go/ --go_opt=paths=source_relative --js_out=public/ efflux.proto

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