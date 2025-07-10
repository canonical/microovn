setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}

