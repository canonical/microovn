# This is a bash shell fragment -*- bash -*-

load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

cluster_register_test_functions() {
    for db in nb sb; do
        bats_test_function \
            --description "OVN ${db^^} DB clustered" \
            -- cluster_test_db_clustered "$db"
        bats_test_function \
            --description "OVN ${db^^} DB connection string" \
            -- cluster_test_db_connection_string "$db"
    done
    bats_test_function \
        --description "Northd connection string" \
        -- cluster_test_northd_connection_string
    bats_test_function \
        --description "Expected MicroOVN cluster count" \
        -- cluster_expected_count
    bats_test_function \
        --description "Expected services up" \
        -- cluster_expected_services_up
    bats_test_function \
        --description "Expected address family for cluster address" \
        -- cluster_expected_address_family
    bats_test_function \
        --description "Open_vSwitch external_ids:ovn-remote addresses" \
        -- cluster_ovs_ovn_remote_addresses
    bats_test_function \
        --description "Ensure northd connectivity between NB and SB" \
        -- cluster_test_southbound_propagation
}

cluster_expected_count() {
    # Extremely simplified check that cluster has required number of members
    local expected_cluster_members=0
    for container in $TEST_CONTAINERS; do
        expected_cluster_members=$(($expected_cluster_members+1))
    done

    for container in $TEST_CONTAINERS; do
        echo "Checking cluster members on $container"
        run lxc_exec "$container" "microovn cluster list --format json | jq length"
        assert_output $expected_cluster_members
    done
}

cluster_expected_services_up() {
    # Check that all expected services are active on cluster members
    local chassis_services="snap.microovn.chassis \
                            snap.microovn.daemon \
                            snap.microovn.switch"
    local central_services="snap.microovn.ovn-ovsdb-server-nb \
                            snap.microovn.ovn-ovsdb-server-sb \
                            snap.microovn.ovn-northd \
                            $chassis_services"

    for container in $TEST_CONTAINERS; do
        local container_services
        container_services=$(microovn_get_cluster_services "$container")
        local check_services
        [[ "$container_services" == *"central"* ]] && \
            check_services=$central_services || \
            check_services=$chassis_services

        for service in $check_services; do
            echo "Checking status of $service on $container"
            run lxc_exec "$container" "systemctl is-active $service"
            assert_output "active"
        done
    done
}

cluster_expected_address_family() {
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

# cluster_test_db_clustered NBSB
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
cluster_test_db_clustered() {
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

cluster_ovs_ovn_remote_addresses() {
    local cluster_addresses=()
    local container_services

    readarray \
        -t cluster_addresses \
        < <(microovn_get_member_cluster_address "central" $TEST_CONTAINERS)
    for container in $TEST_CONTAINERS; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" != *"chassis"* ]]; then
            echo "Skip $container, no chassis services" >&3
            continue
        fi

        run lxc_exec \
            "$container" \
            "microovn.ovs-vsctl get Open_vSwitch . external_ids:ovn-remote"
        for addr in "${cluster_addresses[@]}"; do
            local expected_addr
            expected_addr=$(print_address $addr)
            # By using a fully qualified search string we can safely use
            # partial matching.
            assert_output -p "ssl:$expected_addr:6642"
        done
    done
}

# cluster_test_db_connection_string NBSB
#
# Tests that database connection string for NBSB contains the expected
# addresses.
#
# The NBSB argument can be set to either `nb` or `sb` to indicate which
# database to check.
#
# This test is implemented as a helper function so that we can make use of
# the bats matrix/parallelization capabilities and is kept in this file
# because it performs assertions through the `bats-assert` test_helper
# libraries.
cluster_test_db_connection_string() {
    local nbsb=$1; shift

    local check_var
    [ "$nbsb" == "nb" ] && \
        check_var=OVN_NB_CONNECT || \
        check_var=OVN_SB_CONNECT
    local expected_port
    [ "$nbsb" == "nb" ] && \
        expected_port=6641 || \
        expected_port=6642

    local cluster_addresses=()

    readarray \
        -t cluster_addresses \
        < <(microovn_get_member_cluster_address "central" $TEST_CONTAINERS)
    for container in $TEST_CONTAINERS; do
        run lxc_exec \
            "$container" \
            "grep ^$check_var /var/snap/microovn/common/data/env/ovn.env"
        for addr in "${cluster_addresses[@]}"; do
            local expected_addr
            expected_addr=$(print_address $addr)
            # By using a fully qualified search string we can safely use
            # partial matching.
            assert_output -p "ssl:${expected_addr}:${expected_port}"
        done
    done
}

# cluster_test_northd_connection_string
#
# Test that northd service is connected to all expected NB and SB database
# cluster members.
cluster_test_northd_connection_string() {
    local cluster_addresses=()
    readarray \
        -t cluster_addresses \
        < <(microovn_get_member_cluster_address "central" $TEST_CONTAINERS)

    local container
    for container in $TEST_CONTAINERS; do
        local container_services
        container_services=$(microovn_get_cluster_services)
        if [[ "$container_services" != *"central"* ]]; then
            echo "Skip $container, no central services" >&3
            continue
        fi

        local northd_pid
        northd_pid=$(microovn_get_service_pid "$container" "ovn-northd" "ovn")
        run lxc_exec \
            "$container" \
            "ps -ww -o cmd -p $northd_pid"

        for addr in "${cluster_addresses[@]}"; do
            local expected_addr
            expected_addr=$(print_address $addr)
            # By using a fully qualified search string we can safely use
            # partial matching.
            assert_output -p "ssl:${expected_addr}:6641"
            assert_output -p "ssl:${expected_addr}:6642"
        done
    done
}

# cluster_test_southbound_propagation
#
# Tests that database connection between northbound and southbound databases
# is properly functional.
#
# The command tested is one which waits for the change to propagate to the
# southbound database and was specficially picked as it was at one point
# failing until pointed out by microcloud
cluster_test_southbound_propagation() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-nbctl --timeout=10 --wait=sb ha-chassis-group-add testnet"
        assert_success
        run lxc_exec "$container" \
            "microovn.ovn-nbctl --timeout=10 --wait=sb ha-chassis-group-del testnet"
        assert_success
    done
}

cluster_register_test_functions
