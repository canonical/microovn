setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 1)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS
}

teardown_file() {
    delete_containers $TEST_CONTAINERS
}
