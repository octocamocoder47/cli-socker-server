package_name := $(if $(name),$(name),my_package_name)
# package_name=$1

build_file := $(if $(file),$(file),main.go)
# build_file=$2

init:
	@go mod init ${package_name}
	@$(MAKE) tidy

tidy:
	@go mod tidy

build:
	@mkdir -p bin
	@go build -o ./bin/main $(build_file)

run: build
	@./bin/main
