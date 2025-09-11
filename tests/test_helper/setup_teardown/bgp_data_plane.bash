# setup_file
#
# This functions sets up a simple topology for testing data-plane
# connectivity between external networks and OVN networks advertised
# via BGP.
#
#  +------------------+                +------------------+                +------------------+
#  |     Ext. Host    |     EXT_NET    |  TOR (BGP peer)  |    INT_NET     |    OVN Chassis   |
#  |             eth1 ------------------eth2          eth1------------------eth1              |
#  +------------------+                +------------------+                +------------------+
#
setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/bgp_utils.bash

    TEST_CONTAINER=$(container_names "$BATS_TEST_FILENAME" 1 | tr -d '[:space:]')
    export TEST_CONTAINER

    launch_containers_args "-c linux.kernel_modules=vrf,openvswitch -c security.nesting=true" $TEST_CONTAINER
    wait_containers_ready $TEST_CONTAINER
    install_ppa_netplan $TEST_CONTAINER
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINER
    setup_snap_aliases $TEST_CONTAINER
    bootstrap_cluster $TEST_CONTAINER



    # Setup networks between MicroOVN chassis, BGP peer and external host
    BGP_INT_NET="ovn-bgp-net"
    BGP_EXT_NET="ext-bgp-net"
    create_lxd_network_no_dhcp "$BGP_INT_NET"
    create_lxd_network_no_dhcp "$BGP_EXT_NET"


    # Launch BGP peer container
    BGP_PEER="microovn-bgp-peer"
    launch_containers "$BGP_PEER"
    wait_containers_ready "$BGP_PEER"

    # Launch host on external network
    EXT_HOST="microovn-ext-host"
    launch_containers "$EXT_HOST"
    wait_containers_ready "$EXT_HOST"

    # Connect containers via LXD networks
    BGP_CONTAINER_INT_IFACE="eth1"
    BGP_CONTAINER_EXT_IFACE="eth2"
    OVN_CONTAINER_INT_IFACE="eth1"
    EXT_CONTAINER_EXT_IFACE="eth1"

    connect_container_to_network_no_ip "$BGP_PEER" "$BGP_INT_NET" "$BGP_CONTAINER_INT_IFACE"
    connect_container_to_network_no_ip "$BGP_PEER" "$BGP_EXT_NET" "$BGP_CONTAINER_EXT_IFACE"
    connect_container_to_network_no_ip "$TEST_CONTAINER" "$BGP_INT_NET" "$OVN_CONTAINER_INT_IFACE"
    connect_container_to_network_no_ip "$EXT_HOST" "$BGP_EXT_NET" "$EXT_CONTAINER_EXT_IFACE"


    # Install FRR in peer containers
    install_frr_bgp $BGP_PEER

    # Export test-related variables
    export BGP_PEER
    export EXT_HOST
    export TEST_CONTAINER
    export BGP_CONTAINER_INT_IFACE
    export BGP_CONTAINER_EXT_IFACE
    export EXT_CONTAINER_EXT_IFACE
    export OVN_CONTAINER_INT_IFACE
    export BGP_INT_NET
    export BGP_EXT_NET
}


teardown_file() {
    delete_containers "$TEST_CONTAINER $BGP_PEER $EXT_HOST"
    delete_lxd_network "$BGP_INT_NET"
    delete_lxd_network "$BGP_EXT_NET"
}
