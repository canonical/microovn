# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS and EAST_WEST_ADDRS are populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

@test "Check that custom IP encapsulation works" {
    local container_services
    EAST_WEST_ADDRS=()
    while IFS= read -r line; do
        EAST_WEST_ADDRS+=("$line")
    done < "$BATS_TMPDIR/east_west_addrs.txt"

    for pair in "${EAST_WEST_ADDRS[@]}"; do
        IFS='@' read -r container ip_east_west <<< "$pair"

        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" != *"chassis"* ]]; then
            echo "Skip $container, no chassis services" >&3
            continue
        fi

        run lxc_exec \
            "$container" \
            "microovn.ovs-vsctl get Open_vSwitch . external_ids:ovn-encap-ip"
        assert_output -p $ip_east_west
    done

    local test_filename=tests/init_cluster
    test_filename+=$(test_is_ipv6_test && echo _ipv6 || true)
    test_filename+=.bats

    # Run the cluster test with the custom IP encapsulation
    run bats -F junit $test_filename

    echo "# $output" >&3
    echo "#" >&3
}
