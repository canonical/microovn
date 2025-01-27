MICROOVN_SNAP=microovn.snap
export MICROOVN_SNAP_PATH := $(CURDIR)/$(MICROOVN_SNAP)

ifndef MICROOVN_SNAP_CHANNEL
	export MICROOVN_SNAP_CHANNEL="22.03/stable"
endif

.DEFAULT_GOAL := $(MICROOVN_SNAP)

ALL_TESTS := $(wildcard tests/*.bats)
MICROOVN_SOURCES := $(shell find microovn/ -type f)
COMMAND_WRAPPERS := $(shell find snapcraft/ -type f)
SNAP_SOURCES := $(shell find snap/ -type f)

check: check-lint check-system

check-tabs:
	grep -lrP "\t" tests/ && exit 1 || exit 0

check-lint: check-tabs
	find tests/ \
		-type f \
		-not -name \*.yaml \
		-not -name \*.swp \
		| xargs shellcheck --severity=warning && echo Success!

$(ALL_TESTS): $(MICROOVN_SNAP)
	echo "Running functional test $@";					\
	$(CURDIR)/.bats/bats-core/bin/bats $@

check-system: $(ALL_TESTS)

$(MICROOVN_SNAP): $(MICROOVN_SOURCES) $(SNAP_SOURCES) $(COMMAND_WRAPPERS)
	echo "Building the snap";						\
	snapcraft pack -v -o $(MICROOVN_SNAP)

clean:
	rm -f $(MICROOVN_SNAP_PATH);						\
	snapcraft clean

.PHONY: $(ALL_TESTS) clean check-system check-lint check-tabs
