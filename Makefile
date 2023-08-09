MICROOVN_SNAP=microovn.snap
export MICROOVN_SNAP_PATH := $(CURDIR)/$(MICROOVN_SNAP)
export MICROOVN_TEST_ROOT := $(CURDIR)/tests

ifndef MICROOVN_SNAP_CHANNEL
	export MICROOVN_SNAP_CHANNEL="latest/stable"
endif

check: check-lint check-system

check-tabs:
	grep -lrP "\t" tests/ && exit 1 || exit 0

check-lint: check-tabs
	find tests/ \
		-type f \
		-not -name \*.yaml \
		-not -name \*.swp \
		| xargs shellcheck --severity=warning && echo Success!

check-system: $(MICROOVN_SNAP) generate-tests
	echo "Running functional tests";					\
	bats tests/upgrade.bats

$(MICROOVN_SNAP):
	echo "Building the snap";						\
	snapcraft pack -v -o $(MICROOVN_SNAP)

clean:
	rm -f $(MICROOVN_SNAP_PATH);						\
	snapcraft clean

generate-tests:
	$(MICROOVN_TEST_ROOT)/generate_tests.bash
