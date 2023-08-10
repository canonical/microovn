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
    local cluster_test_filename="${ABS_TOP_TEST_DIRNAME=}test_helper/bats/post_upgrade_cluster.bats"
    local tls_test_filename="${ABS_TOP_TEST_DIRNAME=}test_helper/bats/post_upgrade_tls.bats"

    # Note that the outer bats runner will perform validation on the
    # number of tests ran based on TAP output, so it is important that
    # the inner bats runner uses a different format for its output.
    run bats -F junit $cluster_test_filename

    echo "# $output" >&3
    echo "#" >&3
    assert_success

    run bats -F junit $tls_test_filename

    echo "# $output" >&3
    echo "#" >&3
    assert_success
}
