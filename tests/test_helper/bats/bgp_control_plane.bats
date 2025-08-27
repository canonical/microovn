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

bgp_control_plane_register_test_functions() {
    bats_test_function \
        --description "OVN with multiple BGP peers (Automatic BGP config)" \
        -- bgp_unnumbered_peering yes yes

    bats_test_function \
        --description "OVN with multiple BGP peers (Manual BGP config)" \
        -- bgp_unnumbered_peering no yes

    bats_test_function \
        --description "OVN with single BGP peer (Automatic BGP config)" \
        -- bgp_unnumbered_peering yes no

    bats_test_function \
        --description "OVN with single BGP peer (Manual BGP config)" \
        -- bgp_unnumbered_peering no no

    bats_test_function \
        --description "Enable BGP without configuration" \
        -- bgp_no_config
}

# bgp_unnumbered_peering AUTOCONFIG_BGP MULTI_LINK
#
# This test enables BGP service on each MicroOVN chassis and verifies
# that each chassis can form a BGP neighbor connection with a standalone
# FRR daemon running on a separate host(s).
# This test can be configured via positional arguments to create multiple
# scenarios:
#
# If AUTOCONFIG_BGP is set to "yes", MicroOVN will automatically configure
# FRR services running on OVN chassis to start BGP daemons in the "unnumbered" mode.
# If AUTOCONFIG_BGP is any other value, the FRR's CLI will be used directly to configure
# the BGP daemons.
#
# IF MULTI_LINK is set to "yes", two interfaces per OVN chassis will be used for connection
# with two separate BGP neighbors. IF MULTI_LINK is any other value, only one interface/bgp daemon
# will be used.
#
# Note: For more details about the topology of this test, see comments in the 'setup_teardown/bgp.bash' file.
bgp_unnumbered_peering() {
    local autoconfig_bgp=$1; shift
    local multi_link=$1; shift

    # Populate list of containers on which BGP service gets enabled
    export USED_BGP_CHASSIS=$TEST_CONTAINERS

    local tor_asn=4200000100
    local container
    for container in $BGP_PEERS; do
        tor_asn="$((tor_asn + 1))"
        echo "# Starting BGP in $container on interface $BGP_CONTAINER_IFACE" >&3
        frr_start_bgp_unnumbered "$container" "$BGP_CONTAINER_IFACE" "$tor_asn"
    done

    local host_asn=4210000000
    local i=0
    for container in $TEST_CONTAINERS; do
        local vrf="$((i + 1))0"
        local bgp_iface_1="v$OVN_CONTAINER_NET_1_IFACE-bgp"
        local bgp_iface_2="v$OVN_CONTAINER_NET_2_IFACE-bgp"
        local vrf_device="ovnvrf$vrf"
        host_asn="$((host_asn + 1))"

        # Set up external connection string, used to configure MicroOVN BGP, based on number of upstream
        # links
        local external_connections="$OVN_CONTAINER_NET_1_IFACE"
        if [ "$multi_link" == "yes" ]; then
            external_connections="$external_connections,$OVN_CONTAINER_NET_2_IFACE"
        fi

        # Configure FRR on the OVN chassis either automatically (by supplying ASN) or manually
        # (via FRR CLI)
        if [ "$autoconfig_bgp" == "yes" ]; then
            echo "# Enabling MicroOVN BGP in $container with automatic daemon configuration (ASN $host_asn)" >&3
            lxc_exec "$container" "microovn enable bgp \
                --config ext_connection=$external_connections \
                --config vrf=$vrf \
                --config asn=$host_asn"
        else
            echo "# Enabling MicroOVN BGP in $container with manual daemon configuration" >&3
            lxc_exec "$container" "microovn enable bgp \
                --config ext_connection=$external_connections \
                --config vrf=$vrf"

            echo "# Manually configuring FRR to start BGP daemon on $bgp_iface_1 (ASN $host_asn)" >&3
            microovn_start_bgp_unnumbered "$container" "$bgp_iface_1" "$host_asn" "$vrf_device"
            if [ "$multi_link" == "yes" ]; then
                echo "# Manually configuring FRR to start BGP daemon on $bgp_iface_2 (ASN $host_asn)" >&3
                microovn_start_bgp_unnumbered "$container" "$bgp_iface_2" "$host_asn" "$vrf_device"
            fi
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

        if [ "$multi_link" == "yes" ]; then
            echo "# ($container) waiting on established BGP with $neighbor_2" >&3
            wait_until "microovn_bgp_established $container $vrf_device $neighbor_2"
        fi

        # Set up NAT in OVN
        local nat_ext_ip="172.16.$i.10"
        local nat_logic_ip="192.168.$i.10"
        echo "# ($container) setting up DNAT $nat_ext_ip -> $nat_logic_ip" >&3
        lxc_exec "$container" "microovn.ovn-nbctl lr-nat-add lr-$container-microovn dnat $nat_ext_ip $nat_logic_ip"

        # Wait for the route to external NAT address to show up in BGP peer's routing table
        wait_until "container_has_ipv4_route $neighbor_1 $nat_ext_ip $BGP_CONTAINER_IFACE"

        if [ "$multi_link" == "yes" ]; then
            wait_until "container_has_ipv4_route $neighbor_2 $nat_ext_ip $BGP_CONTAINER_IFACE"
        fi

        i=$((++i))
    done
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

bgp_control_plane_register_test_functions
