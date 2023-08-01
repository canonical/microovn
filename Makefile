MICROOVN_SNAP_PATH=$(CURDIR)/microovn.snap
export MICROOVN_SNAP_PATH

build:
	@echo "Building the snap"
	@snapcraft pack --use-lxd -o $(MICROOVN_SNAP_PATH)

func-tests: build
	@echo "Running functional tests"
	@bats tests/basic_cluster.bats

.PHONY: func-tests build
