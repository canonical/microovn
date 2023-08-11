MICROOVN_RUNDIR=/var/snap/microovn/common/run

function install_microovn() {
    local snap_file=$1; shift
    local containers=$*

    for container in $containers; do
        echo "# Deploying MicroOVN to $container" >&3
        lxc_file_push "$snap_file" "$container/tmp/microovn.snap"
        echo "# Installing MicroOVN in container $container" >&3
        lxc_exec "$container" "snap install /tmp/microovn.snap --dangerous"
        echo "# Connecting plugs in container $container" >&3
        lxc_exec "$container" "for plug in firewall-control \
                                           hardware-observe \
                                           hugepages-control \
                                           network-control \
                                           openvswitch-support \
                                           process-control \
                                           system-trace; do \
                                   sudo snap connect microovn:\$plug;done"
    done
}

function microovn_cluster_get_join_token() {
    local existing_member=$1; shift
    local new_member=$1; shift

    lxc_exec "$existing_member" "microovn cluster add $new_member"
}

function bootstrap_cluster() {
    local leader=""
    local containers=$*

    for container in $containers; do
        if [ -z "$leader" ]; then
            echo "# Bootstrapping MicroOVN on $container" >&3
            lxc_exec "$container" "microovn cluster bootstrap"
            leader="$container"
            continue
        fi

        echo "# Adding $container to the cluster" >&3
        local token
        token=$(lxc_exec "$leader" "microovn cluster add $container")
        echo "# Joining cluster with $container" >&3
        lxc_exec "$container" "microovn cluster join $token"
    done
    wait_ovn_services $containers
}

# wait_ovn_services CONTAINER1 [CONTAINER2 ...]
#
# Wait for OVN "central" services running on the CONTAINERs to start listening.
# Containers with disabled "central" service are skipped.
function wait_ovn_services() {
    local containers=$*
    local ovn_ports="6641 6642 6643 6644"

    for container in $containers; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" != *"central"* ]]; then
            continue
        fi

        echo "# ($container) Waiting for OVN Central services to start"
        for port in $ovn_ports; do
            wait_for_open_port "$container" "$port" 30
        done
    done
}
function microovn_init_create_cluster() {
    local container=$1; shift
    local address=$1; shift

    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "yes\n"
}

expect "Please choose a name for this system" {
    send "\n"
}

expect "Would you like to add additional servers to the cluster?" {
    send "no\n"
}

expect eof
EOF
}

function microovn_init_join_cluster() {
    local container=$1; shift
    local address=$1; shift
    local token=$1; shift
    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "no\n"
}

expect "Please enter your join token:" {
    send "$token\n"
}

expect eof
EOF
}

function microovn_get_cluster_address() {
    local container=$1; shift

    lxc_exec "$container" "microovn status" | \
        awk -F\( "/$container/{sub(/\)\$/,\"\");print\$2}"
}

# _ovn_schema_name NBSB
#
# Print the schema name for NBSB.
#
# Valid values for NBSB are `nb` for Northbound DB or `sb` for Southbound DB.
function _ovn_schema_name() {
    local nbsb=$1; shift

    [ "$nbsb" == "nb" ] \
        && echo OVN_Northbound \
        || echo OVN_Southbound
}

# microovn_ovndb_cluster_status CONTAINER NBSB
#
# Print OVN OVSDB cluster status for database type NBSB from the point of view
# of CONTAINER.
#
# Valid values for NBSB are `nb` for Northbound DB or `sb` for Southbound DB.
function microovn_ovndb_cluster_status() {
    local container=$1; shift
    local nbsb=$1; shift

    local schema_name;
    schema_name=$(_ovn_schema_name "$nbsb")

    lxc_exec "$container" \
             "microovn.ovn-appctl \
                 -t /var/snap/microovn/common/run/ovn/ovn${nbsb}_db.ctl \
                     cluster/status ${schema_name}"
}

# microovn_ovndb_cluster_id CONTAINER NBSB
#
# Print OVN OVSDB cluster ID for database type NBSB from the point of view
# of CONTAINER.
#
# Valid values for NBSB are `nb` for Northbound DB or `sb` for Southbound DB.
function microovn_ovndb_cluster_id() {
    local container=$1; shift
    local nbsb=$1; shift

    local schema_name;
    schema_name=$(_ovn_schema_name "$nbsb")

    lxc_exec "$container" \
             "microovn.ovn-appctl \
                 -t /var/snap/microovn/common/run/ovn/ovn${nbsb}_db.ctl \
                     cluster/cid ${schema_name}"
}

# microovn_get_cluster_services CONTAINER
#
# Print MicroOVN services for CONTAINER from the point of view of CONTAINER.
function microovn_get_cluster_services() {
    local container=$1; shift

    lxc_exec "$container" "microovn status" | \
        grep -A1 "$container" | tr -d ',' | awk -F: '/Services:/{print$2}'
}

# microovn_get_member_cluster_address SERVICE CONTAINER [...]
#
# Print list of cluster addresses by interrogating every member with SERVICE.
#
# Note that we do it this way intentionally to help uncover any
# inconsistencies (as opposed to asking a single member for all the
# addresses).
function microovn_get_member_cluster_address() {
    local service=$1; shift
    local containers=$*

    for container in $containers; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"$service"* ]]; then
            echo "$(microovn_get_cluster_address $container)"
        fi
    done
}

# microovn_get_service_pid CONTAINER SERVICE [ RUNDIR ]
#
# Print PID of MicroOVN service SERVICE running in CONTAINER.
#
# RUNDIR can be one of 'ovn' or 'switch' and defaults to 'ovn'.
function microovn_get_service_pid() {
    local container=$1; shift
    local service=$1; shift
    local rundir=${1:-ovn}

    lxc_exec "$container" "cat ${MICROOVN_RUNDIR}/${rundir}/${service}.pid"
}
