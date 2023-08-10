# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/upgrade_procedures.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}


@test "Verify that currently released snap can be upgraded" {
    echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV" >&3
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    perform_manual_upgrade_steps $TEST_CONTAINERS
    # TODO: include more cluster and TLS tests once we settle on how to include them without copy-pasting
}
