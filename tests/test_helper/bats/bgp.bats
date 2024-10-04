# This is a bash shell fragment -*- bash -*-
load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/bgp_utils.bash

    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
    assert [ -n "$BGP_PEERS" ]
}

# teardown function disables BGP service on any container exported in
# USED_BGP_CHASSIS variable and ensures that the are no leftover
# resources.
teardown() {
    for MICROOVN_BGP_CONTAINER in $USED_BGP_CHASSIS; do
        echo "# ($MICROOVN_BGP_CONTAINER) Disabling MicroOVN BGP" >&3
        lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn disable bgp"

        # Check that OVS Bridges were cleaned up
        run lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn.ovs-vsctl find bridge name!=br-int"
        assert_success
        assert_output ""

        # Check that OVS Ports were cleaned up
        run lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn.ovs-vsctl find interface type!=geneve name!=br-int"
        assert_success
        assert_output ""

        # Check that OVN Bridge mappings were cleaned up
        run lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn.ovs-vsctl get Open_vSwitch . external-ids:ovn-bridge-mappings"
        assert_failure
        assert_output 'ovs-vsctl: no key "ovn-bridge-mappings" in Open_vSwitch record "." column external_ids'
    done

    # Check that NB resources were cleaned up
    run lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn.ovn-nbctl show"
    assert_success
    assert_output ""

    # Restart FRR in peer containers to clean up running config
    local container
    for container in $BGP_PEERS; do
        echo "# ($container) Resetting FRR running configuration" >&3
        lxc_exec "$container" "systemctl restart frr"
    done
}

tls_cluster_register_test_functions() {
    bats_test_function \
        --description "OVN with multiple BGP peers (Automatic BGP config)" \
        -- bgp_multiple_peers yes

    bats_test_function \
        --description "OVN with multiple BGP peers (Manual BGP config)" \
        -- bgp_multiple_peers no

    bats_test_function \
        --description "OVN with single BGP peer (Automatic BGP config)" \
        -- bgp_single_peer yes

    bats_test_function \
        --description "OVN with single BGP peer (Manual BGP config)" \
        -- bgp_single_peer no

    bats_test_function \
        --description "Enable BGP without configuration" \
        -- bgp_no_config
}

# bgp_multiple_peers AUTOCONFIG_BGP
#
# Test that enables BGP on each chassis and configures two
# interfaces per chassis to redirect incoming BGP traffic
# to dedicated <iface_name>-bgp ports.
# If AUTOCONFIG_BGP is set to "yes", MicroOVN will also configure
# FRR BGP daemons to listen on the redirected interfaces.
bgp_multiple_peers() {
    local autoconfig_bgp=$1; shift

    # Populate list of containers on which BGP service gets enabled
    export USED_BGP_CHASSIS=$TEST_CONTAINERS

    local container
    for container in $BGP_PEERS; do
        echo "# Starting BGP in $container on interface $BGP_CONTAINER_IFACE" >&3
        frr_start_bgp_unnumbered "$container" "$BGP_CONTAINER_IFACE" 1
    done

    local i=0
    for container in $TEST_CONTAINERS; do
        local BGP_NET_1_IP="10.$i.10.1/24"
        local BGP_NET_2_IP="10.$i.20.1/24"
        local vrf="$((i + 1))0"
        local bgp_iface_1="$OVN_CONTAINER_NET_1_IFACE-bgp"
        local bgp_iface_2="$OVN_CONTAINER_NET_2_IFACE-bgp"
        local vrf_device="ovnvrf$vrf"
        local asn="1"

        local external_connections="$OVN_CONTAINER_NET_1_IFACE:$BGP_NET_1_IP,$OVN_CONTAINER_NET_2_IFACE:$BGP_NET_2_IP"
        if [ "$autoconfig_bgp" == "yes" ]; then
            echo "# Enabling MicroOVN BGP in $container with automatic daemon configuration (ASN $asn)" >&3
            lxc_exec "$container" "microovn enable bgp \
                --config ext_connection=$external_connections \
                --config vrf=$vrf \
                --config asn=$asn"
        else
            echo "# Enabling MicroOVN BGP in $container with manual daemon configuration" >&3
            lxc_exec "$container" "microovn enable bgp \
                --config ext_connection=$external_connections \
                --config vrf=$vrf"

            echo "# Manually configuring FRR to start BGP daemon on $bgp_iface_1 (ASN $asn)" >&3
            microovn_start_bgp_unnumbered "$container" "$bgp_iface_1" "$asn" "$vrf_device"
            echo "# Manually configuring FRR to start BGP daemon on $bgp_iface_2 (ASN $asn)" >&3
            microovn_start_bgp_unnumbered "$container" "$bgp_iface_2" "$asn" "$vrf_device"
        fi
        # TODO: Figure out why is this sleep necessary
        sleep 3
        i=$((++i))
    done

    local bgp_peers_array
    read -r -a bgp_peers_array <<< $BGP_PEERS
    local i=0
    for container in $TEST_CONTAINERS; do
        local neighbor_1_index=$(($i * 2))
        local neighbor_2_index=$(($i * 2 + 1))
        local neighbor_1=${bgp_peers_array[$neighbor_1_index]}
        local neighbor_2=${bgp_peers_array[$neighbor_2_index]}

        local vrf="$((i + 1))0"
        local vrf_device="ovnvrf$vrf"

        echo "# ($container) waiting on established BGP with $neighbor_1" >&3
        wait_until "microovn_bgp_established $container $vrf_device $neighbor_1"

        echo "# ($container) waiting on established BGP with $neighbor_2" >&3
        wait_until "microovn_bgp_established $container $vrf_device $neighbor_2"

        i=$((++i))
    done
}

# bgp_single_peer AUTOCONFIG_BGP
#
# Test that enables BGP on each chassis and configures single
# interface per chassis to redirect incoming BGP traffic
# to dedicated <iface_name>-bgp port.
# If AUTOCONFIG_BGP is set to "yes", MicroOVN will also configure
# FRR BGP daemon to listen on the redirected interface.
bgp_single_peer() {
    local autoconfig_bgp=$1; shift

    # Populate list of containers on which BGP service gets enabled
    MICROOVN_BGP_CONTAINER=$(echo $TEST_CONTAINERS | awk '{print $1}')
    export USED_BGP_CHASSIS="$MICROOVN_BGP_CONTAINER"

    read -r -a all_bgp_peers <<< "$BGP_PEERS"
    bgp_peer_container="${all_bgp_peers[0]}"
    echo "# Starting BGP in $bgp_peer_container on interface $BGP_CONTAINER_IFACE" >&3
    frr_start_bgp_unnumbered "$bgp_peer_container" "$BGP_CONTAINER_IFACE" 1

    local BGP_NET_IP="10.0.10.1/24"
    local vrf="20"
    local bgp_iface="$OVN_CONTAINER_NET_1_IFACE-bgp"
    local vrf_device="ovnvrf$vrf"
    local asn="1"

    local external_connections="$OVN_CONTAINER_NET_1_IFACE:$BGP_NET_IP"

    if [ "$autoconfig_bgp" == "yes" ]; then
        echo "# Enabling MicroOVN BGP in $MICROOVN_BGP_CONTAINER with automatic daemon configuration (ASN $asn)" >&3
        lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn enable bgp \
            --config ext_connection=$external_connections \
            --config vrf=$vrf \
            --config asn=$asn"
    else
        echo "# Enabling MicroOVN BGP in $MICROOVN_BGP_CONTAINER with manual daemon configuration" >&3
        lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn enable bgp \
            --config ext_connection=$external_connections \
            --config vrf=$vrf"

        echo "# Manually configuring FRR to start BGP daemon on $bgp_iface (ASN $asn)" >&3
        microovn_start_bgp_unnumbered "$MICROOVN_BGP_CONTAINER" "$bgp_iface" "$asn" "$vrf_device"
    fi

    echo "# ($MICROOVN_BGP_CONTAINER) waiting on established BGP with $bgp_peer_container" >&3
    wait_until "microovn_bgp_established $MICROOVN_BGP_CONTAINER $vrf_device $bgp_peer_container"
}

# bgp_no_config
#
# Simple test that enables and disables BGP service without
# any additional configuration.
bgp_no_config() {
    # Populate list of containers on which BGP service gets enabled
    MICROOVN_BGP_CONTAINER=$(echo $TEST_CONTAINERS | awk '{print $1}')
    export USED_BGP_CHASSIS="$MICROOVN_BGP_CONTAINER"

    # Enable BGP without configuring it
    lxc_exec "$MICROOVN_BGP_CONTAINER" "microovn enable bgp"
    # teardown method verifies that it can be properly disabled
}

tls_cluster_register_test_functions
