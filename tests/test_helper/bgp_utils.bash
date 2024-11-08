# install_frr_bgp CONTAINER
#
# install FRR from apt in the CONTAINER and enable BGP service.
function install_frr_bgp() {
    local containers=$*; shift

    local container
    for container in $containers; do
        install_apt_package "$container" frr
        # Enable BGP service in FRR
        lxc_exec "$container" "sed -i 's/bgpd=no/bgpd=yes/g' /etc/frr/daemons"
        # Relax burst restart limit as tests tend to restart the service often
        lxc_exec "$container" "sed -i 's/StartLimitBurst=.*/StartLimitBurst=100/g' /usr/lib/systemd/system/frr.service"
        lxc_exec "$container" "systemctl daemon-reload"
        lxc_exec "$container" "systemctl restart frr"
    done
}

# frr_start_bgp_unnumbered CONTAINER INTERFACE ASN
#
# configure FRR installed via apt in the CONTAINER, to
# start BGP in the unnumbered mode, listening on
# the INTERFACE with ASN.
function frr_start_bgp_unnumbered() {
    local container=$1; shift
    local interface=$1; shift
    local asn=$1; shift

    cat << EOF | lxc_exec "$container" "vtysh"
        configure
        !
        ip prefix-list accept-all seq 5 permit any
        !
        router bgp $asn
        neighbor $interface interface remote-as external
        !
        address-family ipv4 unicast
          neighbor $interface soft-reconfiguration inbound
          neighbor $interface prefix-list accept-all in
        exit-address-family
        !
        address-family ipv6 unicast
          neighbor $interface soft-reconfiguration inbound
          neighbor $interface activate
        exit-address-family
        !
EOF
}

# microovn_start_bgp_unnumbered CONTAINER INTERFACE ASN VRF
#
# configure FRR bundled with MicroOVN in the CONTAINER, to
# start BGP in the unnumbered mode, listening on the INTERFACE
# in the VRF with ASN.
function microovn_start_bgp_unnumbered() {
    local container=$1; shift
    local interface=$1; shift
    local asn=$1; shift
    local vrf=$1; shift

    cat << EOF | lxc_exec "$container" "microovn.vtysh"
        configure
        !
        ip prefix-list no-default seq 5 deny 0.0.0.0/0
        ip prefix-list no-default seq 10 permit 0.0.0.0/0 le 32
        !
        router bgp $asn vrf $vrf
        neighbor $interface interface remote-as external
        !
        address-family ipv4 unicast
          redistribute kernel
          neighbor $interface prefix-list no-default out
        exit-address-family
        !
        address-family ipv6 unicast
          neighbor $interface soft-reconfiguration inbound
          neighbor $interface activate
        exit-address-family
        !
EOF
}

# microovn_bgp_neighbors CONTAINER
#
# Use FRR bundled with the MicroOVN in the CONTAINER to print status
# of its BGP neighbors in the VRF.
function microovn_bgp_neighbors() {
    local container=$1; shift
    local vrf=$1; shift
    echo "$(lxc_exec "$container" "microovn.vtysh -c \"show bgp vrf $vrf neighbors\"")"
}

# microovn_bgp_established CONTAINER VRF NEIGHBOR
#
# Using FRR bundled with MicroOVN in the CONTAINER, return 0
# if BGP daemon running in the VRF successfully established peer
# connection with BGP daemon running on NEIGHBOR host.
function microovn_bgp_established() {
    local container=$1; shift
    local vrf=$1; shift
    local neighbor=$1; shift

    echo "# ($container) Checking BGP established status with neighbor '$neighbor'"
    local status
    status=$(microovn_bgp_neighbors $container $vrf)
    echo "# ($container) Neighbor status: $status"

    grep -A 2 "Hostname: $neighbor$" <<< "$status" | grep "BGP state = Established"
}

#  microovn_bgp_neighbor_address CONTAINER VRF NEIGHBOR
#
# This function logs into CONTAINER and prints the IP address of
# of a BGP NEIGHBOR. The NEIGHBOR parameter is expected to be an interface
# name which is used to set up BGP unnumbered session.
# Since MicroOVN runs BGP daemon in VRF, the VRF name is required as well.
function microovn_bgp_neighbor_address() {
    local container=$1; shift
    local vrf=$1; shift
    local neighbor=$1; shift

    local neighbor_status
    local foreign_host_line
    neighbor_status=$(lxc_exec "$container" "microovn.vtysh -c \"show bgp vrf $vrf neighbor $neighbor\"")
    foreign_host_line=$(grep "^Foreign host:" <<< "$neighbor_status")

    # Print clean IPv6 address of the BGP neighbor
    awk '{print $3}' <<< "$foreign_host_line" | tr -d ','
}
