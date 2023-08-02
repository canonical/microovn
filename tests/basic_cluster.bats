TEST_CONTAINERS=""

setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 3)
    launch_containers jammy jq "${TEST_CONTAINERS[@]}"
    install_microovn "$MICROOVN_SNAP_PATH" "${TEST_CONTAINERS[@]}"
    bootstrap_cluster "${TEST_CONTAINERS[@]}"
}

teardown_file() {
    delete_containers "${TEST_CONTAINERS[@]}"
}

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash
}

@test "Expected MicroOVN cluster count" {
    # Extremely simplified check that cluster has required number of members
    for container in $TEST_CONTAINERS; do
        echo "Checking cluster members on $container"
        run lxc_exec "$container" "microovn cluster list --format json | jq length"
        assert_output "3"
    done
}

@test "Expected services up" {
    # Check that all expected services are active on cluster members
    SERVICES="snap.microovn.central snap.microovn.chassis snap.microovn.daemon snap.microovn.switch"
    for container in $TEST_CONTAINERS; do
        for service in $SERVICES ; do
            echo "Checking status of $service on $container"
            run lxc_exec "$container" "systemctl is-active $service"
            assert_output "active"
        done
    done
}
