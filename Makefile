SHELL=/bin/bash
# Paths to MicroOVN snap
MICROOVN_SNAP=microovn.snap
export MICROOVN_SNAP_PATH := $(CURDIR)/$(MICROOVN_SNAP)

# Name of the LXD image with pre-installed MicroOVN that
# will be used for testing
export MICROOVN_CONTAINER_TEMPLATE := microovn-lxd-template

# If set to "yes", tests will not use template image with pre-installed
# microovn snap. Instead, they will use pristine image and install microovn
# snap manually on each container.
ifndef MICROOVN_TESTS_USE_SNAP
export MICROOVN_TESTS_USE_SNAP := "no"
endif

.DEFAULT_GOAL := $(MICROOVN_SNAP)

ALL_TESTS := $(wildcard tests/*.bats)
MICROOVN_SOURCES := $(shell find microovn/ -type f)
COMMAND_WRAPPERS := $(shell find snapcraft/ -type f)
SNAP_SOURCES := $(shell find snap/ -type f)

export MICROOVN_COVERAGE_DST := $(CURDIR)/.coverage

check: check-lint check-system

check-tabs:
	grep -lrP "\t" tests/ && exit 1 || exit 0

check-lint: check-tabs
	find tests/ \
		-type f \
		-not -name \*.yaml \
		-not -name \*.swp \
		-not -name \*.conf\
		| xargs shellcheck --severity=warning && echo Success!

$(ALL_TESTS): sync-image
	echo "Running functional test $@";					\
	$(CURDIR)/.bats/bats-core/bin/bats $@

check-system: $(ALL_TESTS)

$(MICROOVN_SNAP): $(MICROOVN_SOURCES) $(SNAP_SOURCES) $(COMMAND_WRAPPERS)
	echo "Building the snap";						\
	snapcraft pack -v -o $(MICROOVN_SNAP)

clean:
	rm -f $(MICROOVN_SNAP_PATH);						\
	snapcraft clean

# Create LXD image with MicroOVN snap pre-installed.
$(MICROOVN_CONTAINER_TEMPLATE).tar.gz: $(MICROOVN_SNAP)
	@if [ "$$MICROOVN_TESTS_USE_SNAP" = "yes" ]; then \
		echo "Skipping image build. MICROOVN_TESTS_USE_SNAP is set to 'yes'"; \
	else \
		set -e; \
		source tests/test_helper/lxd.bash; \
		source tests/test_helper/microovn.bash; \
		source tests/test_helper/common.bash; \
		exec 3>&1; \
		base_image="$${MICROOVN_TEST_CONTAINER_IMAGE:-ubuntu:lts}"; \
		echo "Building template image based on $$base_image"; \
		lxc launch -q "$$base_image" $(MICROOVN_CONTAINER_TEMPLATE); \
		wait_containers_ready $(MICROOVN_CONTAINER_TEMPLATE); \
		MICROOVN_TESTS_USE_SNAP=yes install_microovn $(MICROOVN_SNAP_PATH) $(MICROOVN_CONTAINER_TEMPLATE); \
		lxc publish $(MICROOVN_CONTAINER_TEMPLATE) --alias $(MICROOVN_CONTAINER_TEMPLATE) -f --reuse; \
		lxc delete --force $(MICROOVN_CONTAINER_TEMPLATE); \
		lxc image export $(MICROOVN_CONTAINER_TEMPLATE) $(MICROOVN_CONTAINER_TEMPLATE); \
	fi

# Ensure that LXD image used for testing is up to date
sync-image: $(MICROOVN_CONTAINER_TEMPLATE).tar.gz
	@if [ "$$MICROOVN_TESTS_USE_SNAP" = "yes" ]; then \
		echo "Skipping image sync. MICROOVN_TESTS_USE_SNAP is set to 'yes'"; \
	else \
		file_sha=$$(sha256sum $(MICROOVN_CONTAINER_TEMPLATE).tar.gz | awk '{print $$1}' || echo "no file"); \
		image_sha=$$(lxc image info $(MICROOVN_CONTAINER_TEMPLATE) | grep Fingerprint: | awk '{print $$2}' || echo "no image"); \
		if [ "$$file_sha" = "$$image_sha" ]; then \
			echo "MicroOVN image already up to date."; \
		else \
			echo "Uploading MicroOVN template image"; \
			lxc image import $(MICROOVN_CONTAINER_TEMPLATE).tar.gz --alias $(MICROOVN_CONTAINER_TEMPLATE); \
		fi \
	fi

.PHONY: $(ALL_TESTS) clean check-system check-lint check-tabs sync-image
