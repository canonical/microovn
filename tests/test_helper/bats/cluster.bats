# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
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

@test "Expected address family for cluster address" {
    for container in $TEST_CONTAINERS; do
        local addr
        addr=$(microovn_get_cluster_address "$container")
        local test_family
        test_family=$(test_is_ipv6_test && echo inet6 || echo inet)
        local addr_family
        addr_family=$(test_ipv6_addr "$addr" && echo inet6 || echo inet)
        assert [ "$test_family" = "$addr_family" ]
    done
}
