setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 1)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    export MICROOVN_TESTS_USE_SNAP="yes"
    wait_containers_ready $TEST_CONTAINERS
    for TEST_CONTAINER in $TEST_CONTAINERS; do
        lxc_exec "$TEST_CONTAINER" "apt install -y openvswitch-switch"
    done
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
}

teardown_file() {
    print_diagnostics_on_failure $TEST_CONTAINERS
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}
