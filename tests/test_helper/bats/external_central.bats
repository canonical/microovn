# This is a bash shell fragment -*- bash -*-

load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS, INTERNAL_CLUSTER and EXTERNAL_CLUSTER are populated,
    # otherwise the tests below will provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
    assert [ -n "$INTERNAL_CLUSTER" ]
    assert [ -n "$EXTERNAL_CLUSTER" ]
}

external_central_register_test_functions() {
    bats_test_function \
        --description "Configure MicroOVN cluster to connect to the external OVN central" \
        -- configure_microovn_with_external_ovn_central

    bats_test_function \
        --description "Check that central IPs configuration option input is correctly validated" \
        -- check_central_ips_configuration
}

configure_microovn_with_external_ovn_central() {
    # Get IP addresses of containers running the OVN central cluster
    local central_addresses
    local container
    for container in $EXTERNAL_CLUSTER; do
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        if [ -z "$central_addresses" ]; then
            central_addresses="$addr"
        else
            central_addresses="$central_addresses,$addr"
        fi
    done

    local central_containers
    read -r -a central_containers <<< "$EXTERNAL_CLUSTER"
    local datapath_containers
    read -r -a datapath_containers <<< "$INTERNAL_CLUSTER"

    # Configure datapath-only cluster to connect to the external OVN central
    echo "# Configuring ovn.central-ips option: $central_addresses"
    lxc_exec "${datapath_containers[0]}" "microovn config set ovn.central-ips $central_addresses"

    # Ensure that ovn-controllers from datapath cluster registered with the external OVN central
    for container in $INTERNAL_CLUSTER; do
        wait_until "ovn_chassis_registered ${central_containers[0]} $container"
        echo "# Chassis $container successfully registered in the SB database"
    done

    # Ensure that clients in both clusters talk to the same OVN-central
    lxc_exec "${central_containers[0]}" "microovn.ovn-nbctl lr-add R1"
    lxc_exec "${datapath_containers[0]}" "microovn.ovn-nbctl show | grep R1"
}

check_central_ips_configuration() {
    # Get IP addresses of containers running the OVN central cluster
    local addresses
    local container
    for container in $INTERNAL_CLUSTER; do
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        if [ -z "$addresses" ]; then
            addresses="$addr"
        else
            addresses="$addresses,$addr"
        fi
    done

    local containers
    read -r -a containers <<< "$INTERNAL_CLUSTER"

    # Negative tests with unparseable or missing IP addresses
    echo "# Configuring ovn.central-ips with empty value"
    run lxc_exec "${containers[0]}" "microovn config set ovn.central-ips "
    assert_failure

    echo "# Configuring ovn.central-ips with trailing comma"
    run lxc_exec "${containers[0]}" "microovn config set ovn.central-ips $addresses,"
    assert_failure

    echo "# Configuring ovn.central-ips with unparseable IPv4 address"
    run lxc_exec "${containers[0]}" "microovn config set ovn.central-ips 127.0.0"
    assert_failure

    echo "# Configuring ovn.central-ips with unparseable IPv6 address"
    run lxc_exec "${containers[0]}" "microovn config set ovn.central-ips $addresses,[2001:db8::68]"
    assert_failure

    # Positive tests with valid IP addresses
    lxc_exec "${containers[0]}" "microovn config set ovn.central-ips $addresses"
}

external_central_register_test_functions
