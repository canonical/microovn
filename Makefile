SHELL=/bin/bash
MICROOVN_SNAP=microovn.snap
export MICROOVN_SNAP_PATH := $(CURDIR)/$(MICROOVN_SNAP)


export MICROOVN_CONTAINER_TEMPLATE := microovn-lxd-template

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
	set -e;\
	source tests/test_helper/lxd.bash;                   \
	source tests/test_helper/microovn.bash;              \
	source tests/test_helper/common.bash;              \
	exec 3>&1; \
	BATS_TEST_DIRNAME=$(realpath tests/) launch_containers $(MICROOVN_CONTAINER_TEMPLATE);\
	wait_containers_ready $(MICROOVN_CONTAINER_TEMPLATE);\
	install_microovn $(MICROOVN_SNAP_PATH) $(MICROOVN_CONTAINER_TEMPLATE);
	lxc publish $(MICROOVN_CONTAINER_TEMPLATE) --alias $(MICROOVN_CONTAINER_TEMPLATE) -f --reuse
	lxc delete --force $(MICROOVN_CONTAINER_TEMPLATE)
	lxc image export $(MICROOVN_CONTAINER_TEMPLATE) $(MICROOVN_CONTAINER_TEMPLATE)

# Ensure that LXD image used for testing is up to date
sync-image: $(MICROOVN_CONTAINER_TEMPLATE).tar.gz
	file_sha=$$(sha256sum $(MICROOVN_CONTAINER_TEMPLATE).tar.gz | awk '{print $$1}' || echo "no file"); \
	image_sha=$$(lxc image info $(MICROOVN_CONTAINER_TEMPLATE) | grep Fingerprint: | awk '{print $$2}' || echo "no image"); \
	if [ "$$file_sha" == "$$image_sha" ]; then \
		echo "MicroOVN image already up to date."; \
	else \
		echo "Uploading MicroOVN template image"; \
		lxc image import $(MICROOVN_CONTAINER_TEMPLATE).tar.gz --alias $(MICROOVN_CONTAINER_TEMPLATE); \
	fi

.PHONY: $(ALL_TESTS) clean check-system check-lint check-tabs sync-image
