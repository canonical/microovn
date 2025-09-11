setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    export TEST_CONTAINERS
}
