function launch_containers_args() {
    local launch_args=$1; shift
    local containers=$*; shift

    local image="${MICROOVN_TEST_CONTAINER_IMAGE:-ubuntu:lts}"

    for container in $containers; do
        echo "# Launching '$image' container: $container" >&3
        # we actually want word splitting for launch_args to expand multiple
        # arguments and allow empty string.
        # shellcheck disable=SC2086
        lxc launch -q "$image" "$container" $launch_args < \
            "$BATS_TEST_DIRNAME/resources/lxd/instance-config.yaml"
    done
}

function launch_containers() {
    local containers=$*

    launch_containers_args "" "$containers"
}

function launch_vms() {
    local containers=$*; shift

    local image="${MICROOVN_TEST_CONTAINER_IMAGE:-ubuntu:lts}"

    for container in $containers; do
        echo "# Launching '$image' VM: $container" >&3
        lxc launch -q "$image" "$container" --vm
    done
}

function create_lxd_network() {
    local bridge_name=$1; shift

    lxc network create "$bridge_name" ipv6.address=auto > /dev/null 2>&1

    local ipv4_subnet
    local ipv6_subnet

    ipv4_subnet=$(lxc network get "$bridge_name" ipv4.address)
    ipv6_subnet=$(lxc network get "$bridge_name" ipv6.address)

    echo "$ipv4_subnet|$ipv6_subnet"
}

# create_lxd_network_no_dhcp NETWORK_NAME
#
# Create LXC bridge network named NETWORK_NAME without DHCP
function create_lxd_network_no_dhcp() {
    local net_name=$1; shift

    lxc network create "$net_name" -t bridge ipv4.address=none ipv6.address=none
}

function delete_lxd_network() {
    local network_name=$1; shift

    lxc network delete "$network_name"
}

function connect_containers_to_network_ipv4() {
    local containers=$1; shift
    local network_name=$1; shift
    local ipv4_subnet=$1; shift

    local base_ip
    local ip_counter

    base_ip=$(echo $ipv4_subnet | cut -d '/' -f 1)  # Base IP of the subnet
    ip_counter=2  # Start assigning from the second IP in the subnet

    output=""
    for container in $containers; do
        # Add a NIC to the container and attach it to the network
        lxc config device add "${container}" eth1 nic nictype=bridged parent="${network_name}" > /dev/null 2>&1

        # Calculate the next IP address
        local ip="${base_ip%.*}.$ip_counter"
        ((ip_counter++))

        # Get the first interface that is down
        # and configure it with an IPv4 address that
        # is part of the ipv4_subnet
        down_virtIface=$(lxc_exec "${container}" "ip link show | grep 'state DOWN' | awk -F': ' '{print $2}' | head -n 1")
        down_iface="${down_virtIface%@*}"
        lxc_exec "${container}" "ip link set ${down_iface} up && ip addr add $ip/32 dev ${down_iface}"

        output+="$container@$ip,"
    done

    echo "$output"
}

# connect_container_to_network_no_ip CONTAINER NETWORK INTERFACE
#
# add INTERFACE to the CONTAINER that's connected to the lxc
# NETWORK, without setting up any IP addresses on it. The interface
# will keep only its IPv6 LLA address.
function connect_container_to_network_no_ip() {
    local container=$1; shift
    local network=$1; shift
    local interface=$1; shift

    lxc config device add "$container" "$interface" nic \
        name="$interface" nictype=bridged parent="${network}" > /dev/null 2>&1
    lxc exec "$container" ip address flush dev "$interface"
    # Flick interface up and down to retain IPv6 LLA
    lxc exec "$container" ip link set "$interface" down
    lxc exec "$container" ip link set "$interface" up
}

function connect_containers_to_network_ipv6() {
    local containers=$1; shift
    local network_name=$1; shift
    local ipv6_subnet=$1; shift

    local base_ip
    local ip_counter

    base_ip=$(echo "$ipv6_subnet" | sed 's/[^:]*\/.*$//')
    ip_counter=2  # Start assigning from the second IP in the subnet

    output=""
    for container in $containers; do
        # Add a NIC to the container and attach it to the network
        lxc config device add "${container}" eth1 nic nictype=bridged parent="${network_name}" > /dev/null 2>&1

        # Calculate the next IP address
        local ip="${base_ip}$ip_counter"
        ((ip_counter++))

        # Get the first interface that is down
        down_virtIface=$(lxc_exec ${container} "ip link show | grep 'state DOWN' | awk -F': ' '{print $2}' | head -n 1")
        down_iface="${down_virtIface%@*}"
        lxc_exec "${container}" "ip link set ${down_iface} up && ip addr add $ip/128 dev ${down_iface}"

        output+="$container@$ip,"
    done

    echo "$output"
}

# _wait_instance_ready INSTANCE
#
# Wait until cloud-init and snap seeding completed on the LXC
# INSTANCE. This signals that the INSTANCE is ready to be used
function _wait_instance_ready() {
    local instance=$1; shift
    echo "# Waiting for $instance to be ready" >&3
    lxc_exec "$instance" "cloud-init status --wait &&
                           snap wait system seed.loaded"
}

# wait_containers_ready  CONTAINERS
#
# Wait until all CONTAINERS (space separated string of LXC container
# names) are ready to be used. This ensures that cloud-init succeeded
# and snapd is ready to install snaps.
function wait_containers_ready() {
    local containers=$*

    for container in $containers; do
        _wait_instance_ready "$container"
    done
}

# wait_vms_ready  VMS
#
# Wait until all VMS (space separated string of LXC virtual machine
# names) are ready to be used. This ensures that the LXD agent is ready
# to receive `lxc exec` commands, cloud-init succeeded and snapd is ready
# to install snaps.
function wait_vms_ready() {
    local vms=$*
    local vm
    for vm in $vms; do
        wait_until "lxd_agent_ready $vm"
        _wait_instance_ready "$vm"
    done
}

# lxd_agent_ready INSTANCE
#
# This function returns 0 if LXD agent on the INSTANCE is ready, which
# signals that 'lxc exec' can be executed.
# https://discuss.linuxcontainers.org/t/lxd-start-vm-wait/15906/2
function lxd_agent_ready() {
    local instance=$1; shift
    echo "# Waiting for LXC agent on $instance to start" >&3
    ! lxc info "$instance" | grep -q "Processes: -1"
}

function delete_containers() {
    local containers=$*

    for container in $containers; do
        echo "# Cleaning up $container" >&3
        lxc delete --force "$container"
    done
}

function lxc_exec() {
    local container=$1; shift
    local cmd=$1; shift

    lxc exec "$container" -- bash -c "$cmd"

}

function lxc_file_push() {
    local file_path=$1; shift
    local container_path=$1; shift

    lxc file push -q "$file_path" "$container_path"
}

# lxc_file_replace LOCAL_SRC CONTAINER_DST
#
# This function allows pushing files to LXC containers
# even if the target file already exists in the destination.
# LOCAL_SRC argument should be local path to the file to be copied,
# and CONTAINER_DST should be the destination in format
# "<container_name>/path/to/dst"
function lxc_file_replace() {
    local local_src=$1; shift
    local container_dst=$1; shift

    lxc file delete -q "$container_dst" >/dev/null 2>&1 || true
    lxc file push -q "$local_src" "$container_dst"
}

# lxc_pull_dir CONTAINER_PATH DST
#
# Copy directory and all its contents from CONTAINER_PATH to DST.
# CONTAINER_PATH is a source path in form "<container_name>/path/to/file"
# and DST is local destination path to which files will be copied.
function lxc_pull_dir() {
    local container_path=$1; shift
    local dst=$1; shift

    lxc file pull -q --recursive "$container_path" "$dst"
}

# lxc_file_transfer SRC_CONTAINER SRC_PATH DST_CONTAINER DST_PATH
#
# Move file from SRC_PATH on SRC_CONTAINER to DST_PATH on DST_CONTAINER. Since
# the file needs to be temporarily stored on the host running the tests, this function
# needs to have write permission to create temporary folders via mktemp.
# After the transfer, the file is owned by the 'ubuntu' user on the DST_CONTAINER
function lxc_file_transfer() {
    local src_container=$1; shift
    local src_path=$1; shift
    local dst_container=$1; shift
    local dst_path=$1; shift

    local transfer_dir
    transfer_dir=$(mktemp -d)
    echo "# Transferring file from $src_container$src_path"
    lxc file pull "$src_container$src_path" "$transfer_dir"

    echo "# Transferring file to $dst_container$dst_path"
    lxc file push "$transfer_dir/$(basename "$src_path")" "$dst_container$dst_path"

    rm -rf "$transfer_dir"
}

