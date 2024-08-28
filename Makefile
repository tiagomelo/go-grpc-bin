# ==============================================================================
# Help

.PHONY: help
## help: shows this help message
help:
	@ echo "Usage: make [target]\n"
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==============================================================================
# Proto

.PHONY: proto
## proto: compiles .proto files
proto:
	@ docker run --platform linux/amd64 -v $(PWD):/defs namely/protoc-all -i api/proto -f grpcbin.proto -l go -o api/proto/gen/grpcbin --go-source-relative

# ==============================================================================
# Unit tests

.PHONY: test
## test: run unit tests
test:
	@ go test -v ./server/... -count=1

.PHONY: coverage
## coverage: run unit tests and generate coverage report in html format
coverage:
	@ go test -coverprofile=coverage.out ./server/... && go tool cover -html=coverage.out

# ==============================================================================
# Server execution

.PHONY: run
## run: run the gRPC server
run:
	@ if [ -z "$(PORT)" ]; then echo >&2 please set the desired port via the variable PORT; exit 2; fi
	@ go run cmd/main.go -p $(PORT)
