# This is a bash shell fragment -*- bash -*-
load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/bgp_utils.bash

    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure required environment variables are set, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINER" ]
    assert [ -n "$BGP_PEER" ]
    assert [ -n "$EXT_HOST" ]
}

bgp_data_plane_register_test_functions() {
    bats_test_function \
        --description "Test connectivity from External network via TOR to OVN NAT IP" \
        -- ping_ovn_int_network_over_bgp_router

}

function ping_ovn_int_network_over_bgp_router() {
    # Start FRR in BGP peer container
    local tor_asn=4200000100
    echo "# Starting BGP in $BGP_PEER on interface $BGP_CONTAINER_INT_IFACE" >&3
    frr_start_bgp_unnumbered "$BGP_PEER" "$BGP_CONTAINER_INT_IFACE" "$tor_asn"

    # Enable BGP redirection and start BGP daemon in OVN chassis
    local host_asn=4210000000
    local BGP_NET_IP="172.16.10.1/24"
    local vrf="10"
    local vrf_device="ovnvrf$vrf"
    local external_connections="$OVN_CONTAINER_INT_IFACE:$BGP_NET_IP"

    echo "# Enabling MicroOVN BGP in $TEST_CONTAINER and configuring BGP (ASN $host_asn)" >&3
    lxc_exec "$TEST_CONTAINER" "microovn enable bgp \
        --config ext_connection=$external_connections \
        --config vrf=$vrf \
        --config asn=$host_asn"

    echo "# ($TEST_CONTAINER) waiting on established BGP with $BGP_PEER" >&3
    wait_until "microovn_bgp_established $TEST_CONTAINER $vrf_device $BGP_PEER"

    # create VIF that represents VM on the internal OVN network
    local gw_lr="lr-$TEST_CONTAINER-microovn"
    local guest_ls="ls-guest-net"
    local guest_lsp_to_lr="lsp-to-gw"
    local lrp_to_guest_ls="lrp-to-guest"
    local guest_lrp_ip="192.168.10.1"
    local guest_lrp_cidr="$guest_lrp_ip/24"
    local guest_vm_ip="192.168.10.10"
    local guest_vm_cidr="$guest_vm_ip/24"
    local guest_vm_mac="02:00:aa:00:00:01"
    local guest_vm_lsp="lsp-guest-vm"
    local guest_vm_iface="guest-vm"
    local guest_vm_ns="ns-guest"

    echo "# ($TEST_CONTAINER) Create VIF in the OVN network that represents Virtual Machine with IP $guest_vm_cidr" >&3
    echo "# ($TEST_CONTAINER) Create OVN network '$guest_ls' and connect it to router '$gw_lr' ($guest_lrp_cidr)"
    lxc_exec "$TEST_CONTAINER" \
        "microovn.ovn-nbctl \
         -- \
         lrp-add $gw_lr $lrp_to_guest_ls 02:00:ff:00:00:01 $guest_lrp_cidr \
         -- \
         ls-add $guest_ls \
         -- \
         lsp-add $guest_ls $guest_lsp_to_lr \
         -- \
         lsp-set-type $guest_lsp_to_lr router \
         -- \
         lsp-set-options $guest_lsp_to_lr router-port=$lrp_to_guest_ls \
         -- \
         lsp-set-addresses $guest_lsp_to_lr router \
         "

    echo "# ($TEST_CONTAINER) Create LSP '$guest_vm_lsp' ($guest_vm_cidr)"
    lxc_exec "$TEST_CONTAINER" \
        "microovn.ovn-nbctl \
         -- \
         lsp-add $guest_ls $guest_vm_lsp \
         -- \
         lsp-set-addresses $guest_vm_lsp '$guest_vm_mac $guest_vm_cidr'"

    echo "# ($TEST_CONTAINER) Create netns '$guest_vm_ns' and move VM interface '$guest_vm_iface' into it"
    lxc_exec "$TEST_CONTAINER" "ip netns add $guest_vm_ns"
    lxc_exec "$TEST_CONTAINER" \
        "microovn.ovs-vsctl \
         -- \
         add-port br-int $guest_vm_iface \
         -- \
         set Interface $guest_vm_iface type=internal external_ids:iface-id=$guest_vm_lsp"
    netns_ifadd "$TEST_CONTAINER" "$guest_vm_ns" "$guest_vm_iface" "$guest_vm_mac" "$guest_vm_cidr"
    wait_until "microovn_lsp_up $TEST_CONTAINER $guest_vm_lsp"

    echo "# ($TEST_CONTAINER) Set VM's default route via $guest_lrp_ip"
    lxc_exec "$TEST_CONTAINER" "ip netns exec $guest_vm_ns ip route add default via $guest_lrp_ip"

    # Note (mkalcok): Until OVN routers are capable of learning routes advertised
    # by their BGP neighbors, we need to use a crutch in a form static default route
    # via the BGP neighbor's IPv6 LLA
    #
    # Note 2 (mkalcok): It also seems that without specifying egress port for the route, the returning
    # traffic seems to be a hit-or-miss.
    local neighbor_lla
    local egress_port="lrp-$TEST_CONTAINER-$OVN_CONTAINER_INT_IFACE"
    neighbor_lla=$(microovn_bgp_neighbor_address "$TEST_CONTAINER" "$vrf_device" "${OVN_CONTAINER_INT_IFACE}-bgp")
    lxc_exec "$TEST_CONTAINER" "microovn.ovn-nbctl lr-route-add $gw_lr \"0.0.0.0/0\" $neighbor_lla $egress_port"

    # Configure external infrastructure (BGP Peer and External Host)
    echo "# Configure IPv4 networking on the $BGP_EXT_NET"
    local bgp_peer_ext_ip="10.42.0.1"
    local bgp_peer_ext_cidr="$bgp_peer_ext_ip/24"
    local ext_host_ext_ip="10.42.0.10"
    local ext_host_ext_cidr="$ext_host_ext_ip/24"

    lxc_exec "$BGP_PEER" "ip link set $BGP_CONTAINER_EXT_IFACE up && ip addr add $bgp_peer_ext_cidr dev $BGP_CONTAINER_EXT_IFACE"
    lxc_exec "$BGP_PEER" "echo 1 > /proc/sys/net/ipv4/ip_forward"

    lxc_exec "$EXT_HOST" "ip link set $EXT_CONTAINER_EXT_IFACE up && ip addr add $ext_host_ext_cidr dev $EXT_CONTAINER_EXT_IFACE"
    lxc_exec "$EXT_HOST" "ip route del default && ip route add default via $bgp_peer_ext_ip"
    lxc_exec "$EXT_HOST" "ping -W 1 -c 1 $bgp_peer_ext_ip"

    local nat_ext_ip="172.16.10.2"
    echo "# ($TEST_CONTAINER) Create OVN NAT $nat_ext_ip <-> $guest_vm_ip" >&3
    lxc_exec "$TEST_CONTAINER" "microovn.ovn-nbctl lr-nat-add $gw_lr dnat_and_snat $nat_ext_ip $guest_vm_ip"

    # Wait for the route to propagate to BGP peer
    wait_until "container_has_ipv4_route $BGP_PEER $nat_ext_ip $BGP_CONTAINER_INT_IFACE"

    echo "# ($EXT_HOST) Reach NAT address $nat_ext_ip with ping" >&3
    lxc_exec "$EXT_HOST" "ping -W 1 -c 1 $nat_ext_ip"

}
bgp_data_plane_register_test_functions
