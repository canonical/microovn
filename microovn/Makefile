.PHONY: default
default: usage

# Until everyone uses GNU make 4.4.1, which has the --print-tagets option [0],
# we need to list the targets ourselves.
#
# 0: https://git.savannah.gnu.org/cgit/make.git/commit/?id=31036e648f4a92ae0cce215eb3d60a1311a09c60
.PHONY: usage
usage:
	@echo Available targets:
	@echo update-gomod
	@echo update-schema

# Build targets.
.PHONY: build
build:
	go install -v ./cmd/microovn
	go install -v ./cmd/microovnd

# Testing targets.
.PHONY: check
check: check-static check-unit check-system

.PHONY: check-unit
check-unit:
	go test ./...

.PHONY: check-system
check-system:
	true

.PHONY: check-static
check-static:
ifeq ($(shell command -v golangci-lint 2> /dev/null),)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
endif
ifeq ($(shell command -v shellcheck 2> /dev/null),)
	echo "Please install shellcheck"
	exit 1
endif
ifeq ($(shell command -v revive 2> /dev/null),)
	go install github.com/mgechev/revive@latest
endif
	golangci-lint run --timeout 5m
	revive -set_exit_status ./...

# Update targets.
.PHONY: update-gomod
update-gomod:
	go get -u ./...
	go mod tidy

# Update lxd-generate generated database helpers.
.PHONY: update-schema
update-schema:
	go generate ./...
	gofmt -s -w ./database/
	goimports -w ./database/
	@echo "Code generation completed"

