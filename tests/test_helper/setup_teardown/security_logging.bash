setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash

    ABS_TOP_TEST_DIRNAME="${BATS_TEST_DIRNAME}/"
    export ABS_TOP_TEST_DIRNAME

    # This test suite needs to control snap configuration, so it must
    # install the snap itself.
    export MICROOVN_TESTS_USE_SNAP="yes"

    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 1)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS
}

teardown_file() {
    print_diagnostics_on_failure $TEST_CONTAINERS
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}
