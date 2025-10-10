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
          neighbor $interface default-originate
          neighbor $interface soft-reconfiguration inbound
          neighbor $interface prefix-list accept-all in
        exit-address-family
        !
        address-family ipv6 unicast
          neighbor $interface default-originate
          neighbor $interface soft-reconfiguration inbound
          neighbor $interface activate
        exit-address-family
        !
EOF
}

# generate_router_id STRING
#
# generates a random router id in the range 10.0.0.1 - 10.255.255.254 using
# the string as a hash
function generate_router_id() {
    local name="$1"
    local hash
    hash=$(echo -n "$name" | sha256sum | awk '{print $1}')
    echo "10.$(( 0x${hash:0:2} % 256 )).$(( 0x${hash:2:2} % 256 )).$(( 0x${hash:4:2} % 254 + 1 ))"
}

# microovn_bird_apply_default_config CONTAINER
#
# Reset/prepare Bird configuration on the CONTAINER. This function applies basic config
# without any BGP or kernel (vrf) protocols.
function microovn_bird_apply_default_config() {
    local container=$1; shift
    lxc_file_replace "$BATS_TEST_DIRNAME/resources/bird/default.conf" "$container/var/snap/microovn/common/data/bird/bird.conf" 0
    lxc_exec "$container" "microovn.birdc configure"
}

# microovn_bird_add_vrf CONTAINER VRF_TABLE_ID
#
# Add kernel protocol to the Bird configuration on the CONTAINER that learns from
# and exports into the specified VRF.
function microovn_bird_add_vrf() {
    local container=$1; shift
    local vrf_table=$1; shift

    cat << EOF | lxc_exec "$container" "cat >> /var/snap/microovn/common/data/bird/bird.conf"
protocol kernel {
    ipv4 {
        export all;
    };
    learn;
    kernel table $vrf_table;
}
EOF
    lxc_exec "$container" "microovn.birdc configure"

}

# microovn_bird_add_bgp CONTAINER INTERFACE ASN VRF
#
# Add bgp protocol instance to the Bird configuration on the CONTAINER. This instance
# dynamically listens on INTERFACE in VRF and advertises local ASN.
function microovn_bird_add_bgp() {
    local container=$1; shift
    local interface=$1; shift
    local asn=$1; shift
    local vrf=$1; shift

    # BGP connection name in bird can't contain hyphen
    local connection_suffix
    connection_suffix=$(tr "-" "_" <<< "$interface")

    cat << EOF | lxc_exec "$container" "cat >> /var/snap/microovn/common/data/bird/bird.conf"

protocol bgp microovn_$connection_suffix {
    router id $(generate_router_id $container-$interface);
    interface "$interface";
    vrf "$vrf";
    local as $asn;
    neighbor range fe80::/10 external;
    dynamic name "dyn_microovn_$connection_suffix";
    ipv4 {
        next hop self ebgp;
        extended next hop on;
        require extended next hop on;
        import all;
        export filter no_default_v4;
    };
    ipv6 {
        import all;
        export filter no_default_v6;
        };
}
EOF
    lxc_exec "$container" "microovn.birdc configure"
}

# microovn_get_bgp_neighbor_connection_status CONTAINER NEIGHBOR
#
# This function uses Bird client bundled with MicroOVN on the CONTAINER
# to get status of a bgp connection with NEIGHBOR. It prints contents
# of `birdc show protocols all <protocol>` for the protocol where hostname announced
# by the neighbor matches NEIGHBOR.
function microovn_get_bgp_neighbor_connection_status() {
    local container=$1; shift
    local neighbor=$1; shift

    local dyn_connections
    dyn_connections=$(lxc_exec "$container" 'microovn.birdc show protocols \"dyn_microovn_*\"| tail -n +3')

    for connection in $(awk '{print $1}' <<< "$dyn_connections"); do
        connection_details=$(lxc_exec "$container" "microovn.birdc show protocols all $connection")
        if grep "Hostname: $neighbor$" <<< $connection_details; then
            echo "$connection_details"
            return 0
        fi
    done
    return 1
}

# bird_is_bgp_connection_active CONNECTION_DETAILS
#
# This function parses bgp connection described by CONNECTION_DETAILS
# and returns 0 if the state of the connection is "Established" and both
# ipv4 and ipv6 channels are "UP"
#
# CONNECTION_DETAILS is expected to be a string output of
# `birdc show protocol all <protocol>` command.
function bird_is_bgp_connection_active() {
    local connection_details=$1; shift

    local v4_state
    local v6_state

    v4_state=$(grep -A 1 "Channel ipv4" <<< "$connection_details")
    v6_state=$(grep -A 1 "Channel ipv6" <<< "$connection_details")

    grep -qE "BGP state:\s*Established" <<< "$connection_details" \
        && grep -qE "State:\s*UP" <<< "$v4_state" \
        && grep -qE "State:\s*UP" <<< "$v6_state" \

}

# microovn_bgp_established CONTAINER NEIGHBOR
#
# Using Bird bundled with MicroOVN in the CONTAINER, return 0
# if BGP daemon successfully established peer
# connection with BGP daemon running on NEIGHBOR host.
function microovn_bgp_established() {
    local container=$1; shift
    local neighbor=$1; shift

    echo "# ($container) Checking BGP established status with neighbor '$neighbor'"
    local status
    status=$(microovn_get_bgp_neighbor_connection_status "$container" "$neighbor")
    echo "# ($container) Neighbor status: $status"

    bird_is_bgp_connection_active "$status"
}

# microovn_bgp_neighbor_address CONTAINER NEIGHBOR
#
function microovn_bgp_neighbor_address() {
    local container=$1; shift
    local neighbor=$1; shift

    local status
    status=$(microovn_get_bgp_neighbor_connection_status "$container" "$neighbor")

    awk '/Neighbor address/{print$3}' <<< "$status" | cut -f1 -d\%
}
