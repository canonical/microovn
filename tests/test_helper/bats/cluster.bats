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

# _test_db_clustered NBSB
#
# Tests that database is clustered and listens to the expected address.  The
# check is run in all containers that lists `central` as one of its services.
#
# The NBSB argument can be set to either `nb` or `sb` to indicate which
# database to check.
#
# This test is implemented as a helper function so that we can make use of
# the bats matrix/parallelization capabilities and is kept in this file
# because it performs assertions through the `bats-assert` test_helper
# libraries.
function _test_db_clustered() {
    local nbsb=$1; shift

    local cluster_id_str

    for container in $TEST_CONTAINERS; do
        local container_services
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" != *"central"* ]]; then
            echo "Skip $container, no central services" >&3
            continue
        fi

        if [ -z "$cluster_id_str" ]; then
            local cluster_id
            cluster_id=$(microovn_ovndb_cluster_id "$container" "$nbsb")
            local cluster_id_abbrev=
            cluster_id_abbrev=$(echo $cluster_id |cut -c1-4)
            cluster_id_str="Cluster ID: $cluster_id_abbrev ($cluster_id)"
        fi

        echo "Checking DB clustered from ${container}'s point of view" >&3
        local expected_addr
        expected_addr=$(print_address \
            "$(microovn_get_cluster_address $container)")
        local expected_port
        [ "$nbsb" == "nb" ] && expected_port=6643 || expected_port=6644

        run microovn_ovndb_cluster_status "$container" "$nbsb"

        assert_success
        assert_line "$cluster_id_str"
        assert_line "Address: ssl:${expected_addr}:${expected_port}"
        assert_line "Status: cluster member"
    done
}

@test "OVN Northbound DB clustered" {
    # Check Northbound database clustered using expected address/protocol.
    _test_db_clustered nb
}

@test "OVN Southbound DB clustered" {
    # Check Southbound database clustered using expected address/protocol.
    _test_db_clustered sb
}
