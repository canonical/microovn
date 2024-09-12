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

    lxc_exec "$container" "vtysh \
        -c \"configure\" \
        -c \"router bgp $asn\" \
        -c \"neighbor $interface interface remote-as internal\""
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

    lxc_exec "$container" "microovn.vtysh \
        -c \"configure\" \
        -c \"router bgp $asn vrf $vrf\" \
        -c \"neighbor $interface interface remote-as internal\""
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
