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

