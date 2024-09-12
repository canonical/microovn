# This is a bash shell fragment -*- bash -*-

setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 1)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]

    # Trim trailing whitespace from a variable with only single container
    TEST_CONTAINER="$(echo -e "${TEST_CONTAINERS}" | sed -e 's/[[:space:]]*$//')"
    export TEST_CONTAINER
}

teardown() {
    lxc_exec "$TEST_CONTAINER" "snap remove microovn" || true
}

@test "Cleanup OVS datapaths on snap removal" {
    # Verify that removal of MicroOVN snap cleans up DP resources

    # The tests will need external `ovs-dpctl` command to check DPs after
    # microovn removal.
    install_apt_package "$TEST_CONTAINER" "openvswitch-switch"

    echo "Checking datapaths on container '$TEST_CONTAINER' before MicroOVN installation."
    run lxc_exec "$TEST_CONTAINER" "ovs-dpctl dump-dps | wc -l"
    assert_output "0"

    install_microovn "$MICROOVN_SNAP_PATH" "$TEST_CONTAINER"
    bootstrap_cluster "$TEST_CONTAINER"

    echo "Checking datapaths on container '$TEST_CONTAINER' after MicroOVN bootstrap."
    run lxc_exec "$TEST_CONTAINER" "ovs-dpctl dump-dps | wc -l"
    assert_output "1"

    echo "Removing MicroOVN snap."
    run lxc_exec "$TEST_CONTAINER" "snap remove microovn"

    echo "Checking datapaths on container '$TEST_CONTAINER' after MicroOVN removal."
    run lxc_exec "$TEST_CONTAINER" "ovs-dpctl dump-dps | wc -l"
    assert_output "0"
}
