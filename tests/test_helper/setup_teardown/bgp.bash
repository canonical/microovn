# setup_file
#
# This function runs once per test-suite and configures following topology for
# BGP tests:
#        +--------------+             +-------------------+
#        | bgp-peer-1-1 ---------------eth1     microovn-1|
#        +--------------+             |                   |
#        +--------------+             |                   |
#        | bgp-peer-1-2 ---------------eth2          eth0 -------|
#        +--------------+             +-------------------+      |
#                                                                |
#                                                                |
#        +--------------+             +-------------------+      |
#        | bgp-peer-2-1 ---------------eth1     microovn-2|      |
#        +--------------+             |                   |      |
#        +--------------+             |                   |      |
#        | bgp-peer-2-2 ---------------eth2          eth0 --------
#        +--------------+             +-------------------+      |
#                                                                |
#                                                                |
#        +--------------+             +-------------------+      |
#        | bgp-peer-3-1 ---------------eth1     microovn-3|      |
#        +--------------+             |                   |      |
#        +--------------+             |                   |      |
#        | bgp-peer-3-2 ---------------eth2          eth0 -------|
#        +--------------+             +-------------------+
#
# Afterwards, it's up to the tests themselves to enable and
# configure individual BGP services on MicroOVN and Peer nodes.
#
# Note: For the sake of brevity, the names in the image do not match
# with the real container names. Please see BGP_PEERS and TEST_CONTAINERS
# variables for the exact names.
setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/bgp_utils.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 3)
    export TEST_CONTAINERS
    launch_containers_args "-c linux.kernel_modules=vrf,openvswitch" $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS


    # Setup networks for BGP neighbors (format: "ovn-bgp-net-<chassis>-<link>)
    BGP_NETS="\
        ovn-bgp-net-1-1 \
        ovn-bgp-net-1-2 \
        ovn-bgp-net-2-1 \
        ovn-bgp-net-2-2 \
        ovn-bgp-net-3-1 \
        ovn-bgp-net-3-2 \
        "
    local bgp_net
    for bgp_net in $BGP_NETS; do create_lxd_network_no_dhcp "$bgp_net"; done

    # Launch containers for BGP peers (format: microovn-bgp-peer-<chassis>-<peer_id>)
    BGP_PEERS="\
    microovn-bgp-peer-1-1 \
    microovn-bgp-peer-1-2 \
    microovn-bgp-peer-2-1 \
    microovn-bgp-peer-2-2 \
    microovn-bgp-peer-3-1 \
    microovn-bgp-peer-3-2 \
    "

    launch_containers $BGP_PEERS
    wait_containers_ready $BGP_PEERS

    # Connect containers via BGP networks
    BGP_CONTAINER_IFACE="eth1"
    OVN_CONTAINER_NET_1_IFACE="eth1"
    OVN_CONTAINER_NET_2_IFACE="eth2"

    local container
    local bgp_nets_array
    read -r -a bgp_nets_array <<< $BGP_NETS
    local i=0
    for container in $BGP_PEERS; do
        connect_container_to_network_no_ip "$container" "${bgp_nets_array[$i]}" \
            "$BGP_CONTAINER_IFACE"
        i=$((++i))
    done

    i=0
    for container in $TEST_CONTAINERS; do
        local chassis_net_1_index=$(($i * 2))
        local chassis_net_2_index=$(($i * 2 + 1))
        connect_container_to_network_no_ip "$container" \
            "${bgp_nets_array[$chassis_net_1_index]}" \
            "$OVN_CONTAINER_NET_1_IFACE"
        connect_container_to_network_no_ip "$container" \
            "${bgp_nets_array[$chassis_net_2_index]}" \
            "$OVN_CONTAINER_NET_2_IFACE"
        i=$((++i))
    done

    # Install FRR in peer containers
    install_frr_bgp $BGP_PEERS

    # Export BGP-related variables
    export BGP_PEERS
    export BGP_CONTAINER_IFACE
    export OVN_CONTAINER_NET_1_IFACE
    export OVN_CONTAINER_NET_2_IFACE
    export BGP_NETS
}

teardown_file() {
    delete_containers $TEST_CONTAINERS
    delete_containers $BGP_PEERS
    local net
    for net in $BGP_NETS; do
        delete_lxd_network "$net"
    done
}
