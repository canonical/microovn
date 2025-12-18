MICROOVN_RUNDIR=/var/snap/microovn/common/run

function install_microovn() {
    local snap_file=$1; shift
    local containers=$*

    if [ "$MICROOVN_TESTS_USE_SNAP" != "yes" ]; then
        echo "# Skipping snap installation. MicroOVN snap is expected to be present on test containers" >&3
        return 0
    fi

    local snap_base
    snap_base=$(snap_print_base $snap_file)

    for container in $containers; do
        if ! test_snap_is_stable_base "$snap_base"; then
            echo "# !!NOTE!! Installing $snap_base \
                  from edge channel for $snap_file" >&3
            lxc_exec "$container" "snap install --edge $snap_base"
        fi
        echo "# Deploying MicroOVN to $container" >&3
        lxc_file_push "$snap_file" "$container/tmp/microovn.snap"
        echo "# Installing MicroOVN in container $container" >&3
        lxc_exec "$container" "snap install /tmp/microovn.snap --dangerous"
        echo "# Connecting plugs in container $container" >&3
        # Give it few retries as snap can't connect plugs while services
        # are still starting
        local i
        for (( i = 0; i < 10; i++ )); do
            if lxc_exec "$container" "for plug in firewall-control \
                                                  hardware-observe \
                                                  hugepages-control \
                                                  network-control \
                                                  hardware-observe \
                                                  openvswitch-support \
                                                  process-control \
                                                  system-trace \
                                                  network-setup-control; do \
                                          sudo snap connect microovn:\$plug;done"; then
                break
            fi

            if [ "$i" -eq 9 ]; then
                echo "Failed to connect MicroOVN plugs."
                return 1
            fi

            sleep 1
        done
    done
}

# install_microovn_from_store CHANNEL CONTAINER1 [CONTAINER2 ...]
#
# Install MicroOVN snap from specified CHANNEL from Snap store in all CONTAINERs. If
# the CHANNEL argument is an empty string, a default channel will be used.
function install_microovn_from_store() {
    local channel=$1; shift
    local containers=$*
    local source_channel=""
    local channel_pretty_name="default"

    if [ -n "$channel" ]; then
        channel_pretty_name="$channel"
        source_channel="--channel $channel"
    fi
    for container in $containers; do
        echo "# Installing MicroOVN from SnapStore ('$channel_pretty_name' channel) in container $container" >&3
        lxc_exec "$container" "snap install microovn $source_channel"
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
    local custom_encapsulation_ip=$1; shift
    local user_ca_crt=$1; shift
    local user_ca_key=$1; shift
    local services=$1; shift

    custom_encapsulation_ip_dialog=""
    if [ -n "$custom_encapsulation_ip" ]; then
        custom_encapsulation_ip_dialog=$(cat <<EOF
expect "Would you like to define a custom encapsulation IP address for this member?" {
    send "yes\n"
}

expect "Please enter the custom encapsulation IP address for this member" {
    send "$custom_encapsulation_ip\n"
}
EOF
)
    else
        custom_encapsulation_ip_dialog=$(cat <<EOF
expect "Would you like to define a custom encapsulation IP address for this member?" {
    send "\n"
}
EOF
)
    fi

    local user_ca_dialog=""
    if [ -z "$user_ca_crt" ] || [ -z "$user_ca_key" ]; then
        echo "# Automatically generating CA" >&3
        user_ca_dialog=$(cat <<EOF
expect "Would you like to provide your own CA certificate and private key for issuing OVN TLS certificates?" {
    send "\n"
}
EOF
)
    else
        echo "# Using CA supplied by the user" >&3
        user_ca_dialog=$(cat <<EOF
expect "Would you like to provide your own CA certificate and private key for issuing OVN TLS certificates?" {
    send "yes\n"
}

expect "Please enter the path to the CA certificate file:" {
    send "$user_ca_crt\n"
}

expect "Please enter the path to the CA private key file:" {
    send "$user_ca_key\n"
}
EOF
)
    fi

    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "yes\n"
}

expect "Please select comma-separated list services you would like to enable on this node (central/chassis/switch) or let MicroOVN automatically decide (auto)" {
    send "$services\n"
}

expect "Please choose a name for this system" {
    send "$container\n"
}

$custom_encapsulation_ip_dialog

$user_ca_dialog

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
    local custom_encapsulation_ip=$1; shift
    local services=$1; shift

    custom_encapsulation_ip_dialog=""
    if [ -n "$custom_encapsulation_ip" ]; then
        custom_encapsulation_ip_dialog=$(cat <<EOF
expect "Would you like to define a custom encapsulation IP address for this member?" {
    send "yes\n"
}

expect "Please enter the custom encapsulation IP address for this member" {
    send "$custom_encapsulation_ip\n"
}
EOF
)
    else
        custom_encapsulation_ip_dialog=$(cat <<EOF
expect "Would you like to define a custom encapsulation IP address for this member?" {
    send "\n"
}
EOF
)
    fi

    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "no\n"
}

expect "Please select comma-separated list services you would like to enable on this node (central/chassis/switch) or let MicroOVN automatically decide (auto)" {
    send "$services\n"
}

expect "Please enter your join token:" {
    send "$token\n"
}

$custom_encapsulation_ip_dialog

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

# microovn_ovndb_server_id CONTAINER NBSB
#
# Print (short) cluster ID of OVN OVSDB cluster member
#
# Valid values for NBSB are `nb` for Northbound DB or `sb` for Southbound DB.
function microovn_ovndb_server_id() {
    local container=$1; shift
    local nbsb=$1; shift

    local schema_name;
    schema_name=$(_ovn_schema_name "$nbsb")

    local full_sid=""
    full_sid=$(lxc_exec "$container" \
               "microovn.ovn-appctl \
                   -t /var/snap/microovn/common/run/ovn/ovn${nbsb}_db.ctl \
                       cluster/sid ${schema_name}")
    echo "${full_sid:0:4}"
}

# microovn_wait_ovndb_state CONTAINER NBSB STATE TIMEOUT
#
# From the point of view of CONTAINER, wait until the 'nb' or 'sb' database as
# represented by NBSB reaches STATE, waiting a maximum of TIMEOUT seconds.
function microovn_wait_ovndb_state() {
    local container=$1; shift
    local nbsb=$1; shift
    local state=$1; shift
    local timeout=$1; shift

    local schema_name
    schema_name=$(_ovn_schema_name "$nbsb")

    lxc_exec "$container" \
        "timeout ${timeout} microovn.ovsdb-client wait \
         unix:/var/snap/microovn/common/run/ovn/ovn${nbsb}_db.sock \
         ${schema_name} ${state}"
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

    local pid
    pid=$(lxc_exec "$container" \
              "cat ${MICROOVN_RUNDIR}/${rundir}/${service}.pid")

    # Ensure we actually got a PID
    local re='^[0-9]+$'
    [[ "$pid" =~ $re ]] && echo $pid || return 1

}

# microovn_wait_for_service_starttime CONTAINER SERVICE [ RUNDIR ]
#
# Wait until SERVICE has started in CONTAINER and print its pid.
#
# RUNDIR can be one of 'ovn' or 'switch' and defaults to 'ovn'.
function microovn_wait_for_service_starttime() {
    local container=$1; shift
    local service=$1; shift
    local rundir=${1:-ovn}

    local pid
    pid=$(wait_until "microovn_get_service_pid $container $service $rundir")
    get_pid_start_time $container $pid
}

# wait_microovn_online CONTAINER MAX_RETRY
#
# Wait until all MicroOVN members reach online state from the point
# of view of the CONTAINER. There's a 1 second delay between retry attempts
# so MAX_RETRY parameter roughly corresponds to how many seconds it takes for this
# function to time out.
#
# If cluster members do not reach "online" state before the MAX_RETRY is reached, this
# function returns 1 as a return code.

function wait_microovn_online(){
    local container=$1; shift
    local max_retry=$1; shift
    local rc=1

    # Retry with 1s backoff until all MicroOVN members show ONLINE status
    for (( i = 1; i <= "$max_retry"; i++ )); do
        local all_online=1
        echo "# ($container) Waiting for MicroOVN cluster to come ONLINE ($i/$max_retry)"

        # Each line in the output of command below shows individual cluster member status
        run lxc_exec "$container" "microovn cluster list -f json | jq -r .[].status"

        # Fail this iteration if 'microovn cluster list' fails.
        if [ "$status" -ne 0 ]; then
            all_online=0
        fi

        # Parse lines in the command output and fail this iteration if not all lines match
        # the expected member status
        # shellcheck disable=SC2154 # Variable "$output" is exported from previous execution of 'run'
        while read -r status ; do
            if [ "$status" != "ONLINE" ]; then
                echo "# ($container) At least one member in state '$status'"
                all_online=0
            fi
        done <<< "$output"

        if [ $all_online -eq 1 ] ; then
            echo "# ($container) All cLuster members reach ONLINE state"
            rc=0
            break
        fi
        sleep 1
    done

    return $rc

}

# wait_ovsdb_cluster_changes_applied CONTAINER CONTROL_PATH DB_NAME TIMEOUT
#
# Wait until OVN OVSDB cluster member converges with rest of the cluster. This function
# checks output of ovn-appctl to make sure that field 'Entries not yet applied' reaches 0.
# It requires CONTROL_PATH which points to database's .ctl file (i.e. path/to/ovnnb_db.ctl)
# and DB_NAME which should be either "OVN_Northbound" or "OVN_Southbound.
#
# TIMEOUT in seconds is roughly obeyed. If conditions are not met before timeout is reached, this
# functions returns non-zero RC
function wait_ovsdb_cluster_changes_applied() {
    local container=$1; shift
    local ctl_path=$1; shift
    local db_name=$1; shift
    local timeout=$1; shift
    local rc=1
    local retries=""
    retries=$((timeout * 2))

    for (( i = 1; i <= "$retries"; i++ )); do
        echo "# ($container) Waiting for $db_name to apply all changes ($i/$retries)"
        run lxc_exec "$container" "microovn.ovn-appctl -t $ctl_path cluster/status $db_name"
        # shellcheck disable=SC2154 # Variable "$output" is exported from previous execution of 'run'
        echo "# ($container) Cluster status: $output"
        if [[ "$output" == *"Entries not yet applied: 0"* ]]; then
            echo "# ($container) All changes applied to $db_name"
           rc=0
           break
        fi
        sleep 0.5
    done

    return $rc
}

# wait_ovsdb_cluster_container_leave SERVER_ID CONTROL_PATH DB_NAME TIMEOUT CONTAINER1 [CONTAINER2 ...]
#
# Wait until all CONTAINERs confirm that cluster member with SERVER_ID is no longer present in cluster.
#
# This function requires CONTROL_PATH which points to database's .ctl file (i.e. path/to/ovnnb_db.ctl)
# and DB_NAME which should be either "OVN_Northbound" or "OVN_Southbound.
#
# TIMEOUT in seconds is roughly obeyed. If conditions are not met before timeout is reached, this
# functions returns non-zero RC
function wait_ovsdb_cluster_container_leave() {
    local target_server_id=$1; shift
    local ctl_path=$1; shift
    local db_name=$1; shift
    local timeout=$1; shift
    local monitor_containers=$*
    local rc=1
    local retries=""
    retries=$((timeout * 2))

    for (( i = 1; i <= "$retries"; i++ )); do
        local container=""
        local server_present=0
        for container in $monitor_containers; do
            local connection_list=""
            echo "# ($container) Waiting for $target_server_id to depart cluster ($i/$retries)" >&3
            run lxc_exec "$container" "microovn.ovn-appctl -t $ctl_path cluster/status $db_name"
            # shellcheck disable=SC2154 # Variable "$output" is exported from previous execution of 'run'
            echo "# ($container) Status: $output"
            connection_list=$(grep -E '^Connections:' <<< "$output")
            if [[ $connection_list == *"$target_server_id"* ]]; then
                echo "# ($container) Server $target_server_id still present" >&3
                ((++server_present))
            fi
        done

        if [ "$server_present" -eq 0 ]; then
            echo "# Server $target_server_id successfully departed." >&3
            rc=0
            break
        fi
        sleep 0.5
    done

    return $rc
}

# wait_ovsdb_cluster_container_join SERVER_ID CONTROL_PATH DB_NAME TIMEOUT CONTAINER1 [CONTAINER2 ...]
#
# Wait until all CONTAINERs confirm that cluster member with SERVER_ID has successfully joined the cluster.
#
# This function requires CONTROL_PATH which points to the database's .ctl file (i.e. path/to/ovnnb_db.ctl)
# and DB_NAME which should be either "OVN_Northbound" or "OVN_Southbound."
#
# TIMEOUT in seconds is roughly obeyed. If conditions are not met before the timeout is reached, this
# function returns a non-zero RC.
function wait_ovsdb_cluster_container_join() {
    local target_server_id=$1; shift
    local ctl_path=$1; shift
    local db_name=$1; shift
    local timeout=$1; shift
    local monitor_containers=$*
    local rc=1
    local retries=""
    retries=$((timeout * 2))

    for (( i = 1; i <= "$retries"; i++ )); do
        local container=""
        local server_missing=0
        for container in $monitor_containers; do
            local connection_list=""
            echo "# ($container) Waiting for $target_server_id to join cluster ($i/$retries)" >&3
            run lxc_exec "$container" "microovn.ovn-appctl -t $ctl_path cluster/status $db_name"
            # shellcheck disable=SC2154 # Variable "$output" is exported from previous execution of 'run'
            echo "# ($container) Status: $output"
            connection_list=$(grep -E '^Connections:' <<< "$output")
            if [[ $connection_list != *"$target_server_id"* ]]; then
                ((++server_missing))
                echo "# ($container) Server $target_server_id not yet present" >&3
            fi
        done

        if [ "$server_missing" -eq 0 ]; then
            echo "# Server $target_server_id successfully joined." >&3
            rc=0
            break
        fi
        sleep 0.5
    done

    return $rc
}

# microovn_status_is_schema_ok CONTAINER NBSB
#
# Checks whether schema for NBSB (nb|sb) on CONTAINER is OK from the
# perspective of the `microovn status` command.
function microovn_status_is_schema_ok() {
    local container=$1; shift
    local nbsb=$1; shift

    local schema_name
    schema_name=$(_ovn_schema_name "$nbsb")

    local cmd
    printf -v cmd 'microovn status | grep -q %s:\ OK' "${schema_name//_/\\ }"

    lxc_exec "$container" "$cmd"
}

MICROOVN_PREFIX_LS=sw
MICROOVN_PREFIX_LR=lr
MICROOVN_PREFIX_LRP=lrp-sw

MICROOVN_SUFFIX_LRP_LSP=lrp

# microovn_extract_ctn_n CONTAINER
#
# When CONTAINER is a string ending with "-n" this function will extract and
# validate that n is an integer before printing it.
#
# Note that it is up to the caller to ensure that the prerequisite mentioned
# above is met.
function microovn_extract_ctn_n() {
    local container=$1; shift

    local n=${container##*-}
    assert test "$n" -ge 0

    echo "$n"
}

# microovn_add_gw_router CONTAINER
#
# Add LR with options:chassis set to CONTAINER, attach it to LS for use by VIFs
# on CONTAINER.
#
# Name of OVN objects is generated can be influenced using the
# MICROOVN_PREFIX_LS, MICROOVN_PREFIX_LR, MICROOVN_PREFIX_LRP and
# MICROOVN_SUFFIX_LRP_LSP variables.
function microovn_add_gw_router() {
    local container=$1; shift

    local n
    n=$(microovn_extract_ctn_n "$container")
    assert test "$n" -le 255
    local ls_name="${MICROOVN_PREFIX_LS}-${container}"
    local lr_name="${MICROOVN_PREFIX_LR}-${container}"
    local lrp_name="${MICROOVN_PREFIX_LRP}-${container}"
    local lrp_lsp_name="${ls_name}-${MICROOVN_SUFFIX_LRP_LSP}"
    local lrp_addresses
    printf -v lrp_addresses "00:00:02:00:00:%02x 10.42.%d.1/24" "$n" "$n"

    lxc_exec "$container" \
        "microovn.ovn-nbctl \
         -- \
         ls-add $ls_name \
         -- \
         lr-add $lr_name \
         -- \
         set Logical_Router $lr_name options:chassis=$container \
         -- \
         lrp-add $lr_name $lrp_name $lrp_addresses \
         -- \
         lsp-add $ls_name $lrp_lsp_name \
         -- \
         lsp-set-type $lrp_lsp_name router \
         -- \
         lsp-set-options $lrp_lsp_name router-port=$lrp_name \
         -- \
         lsp-set-addresses $lrp_lsp_name router"

}

# microovn_delete_gw_router CONTAINER
#
# Delete LR and LS for CONTAINER previously created by the
# ``microovn_add_gw_router`` function.
function microovn_delete_gw_router() {
    local container=$1; shift

    local ls_name="${MICROOVN_PREFIX_LS}-${container}"
    local lr_name="${MICROOVN_PREFIX_LR}-${container}"

    lxc_exec "$container" \
        "microovn.ovn-nbctl \
         -- \
         lr-del $lr_name \
         -- \
         ls-del $ls_name"

}

# microovn_lsp_up CONTAINER LSP
#
# Use CONTAINER to check status of LSP from OVN Northbound DB point of view.
#
# Returns success if LSP has status 'up', failure otherwise.
function microovn_lsp_up() {
    local container=$1; shift
    local lsp=$1; shift

    result=$(lxc_exec "$container" \
        "microovn.ovn-nbctl --format=table --no-headings \
         find Logical_Switch_Port name=${lsp} up=true | wc -l")
    test "$result" -eq 1
}

# microovn_mac_binding_exists CONTAINER IP LOGICAL_PORT
#
# Use CONTAINER to check existence of Mac_Binding from OVN Southbound DB.
#
# Returns success if exactly one record is found.
function microovn_mac_binding_exists() {
    local container=$1; shift
    local ip=$1; shift
    local logical_port=$1; shift

    local result
    result=$(lxc_exec "$container" \
        "microovn.ovn-sbctl --bare --columns _uuid \
         find Mac_Binding ip='\"${ip}\"' logical_port=${logical_port} | wc -l")
    test "$result" -eq 1
}

# microovn_learned_route_exists CONTAINER IP NEXTHOP [ NEXTHOP ... ]
#
# Use CONTAINER to find one Learned_Route record per pair of IP and NEXTHOP.
function microovn_learned_route_exists() {
    local container=$1; shift
    local ip=$1; shift
    local nexthop="$*"

    local n_results
    n_results=0
    local subcmds
    subcmds=""

    # The *ctl commands can execute multiple commands as an atomic operation by
    # dividing them with --.
    #
    # Note that it is not possible to pass additional arguments to commands
    # such as find in this mode, but that is what we have awk for!
    for nh in $nexthop; do
        ((n_results++))
        subcmds="${subcmds} -- find Learned_Route ip='\"${ip}\"' \
                                    nexthop='\"${nh}\"'"
    done

    local result
    result=$(lxc_exec "$container" \
        "microovn.ovn-sbctl $subcmds | awk '/^_uuid/{print$3}' | wc -l")
    test "$result" -eq $n_results
}

# microovn_add_vif CONTAINER NS_NAME IF_NAME [LS_NAME [LSP_IP]]
#
# Create LSP in LS for CONTAINER, create OVS internal interface with IF_NAME,
# attach it to LSP and move it into network namespace NS_NAME.
#
# LS name is either automatically generated based on CONTAINER name and
# MICROOVN_PREFIX_LS, or it can be optionally specified with LS_NAME argument.
#
# IP address for the LSP/VIF can be optionally provided as LSP_IP argument (in
# CIDR format). Otherwise it will be automatically generated (along with MAC
# address) based on integer found after the last ``-`` in CONTAINER which needs
# to be a numeric value ranging from 0 to 255.
function microovn_add_vif() {
    local container=$1; shift
    local ns_name=$1; shift
    local if_name=$1; shift
    local ls_name=""; if [ "$#" -gt 0 ]; then ls_name="$1"; shift; fi
    local cidr=""; if [ "$#" -gt 0 ]; then cidr="$1"; shift; fi

    local n
    n=$(microovn_extract_ctn_n "$container")
    assert test "$n" -le 255

    if [ -z "$ls_name" ]; then
        ls_name="${MICROOVN_PREFIX_LS}-${container}"
    fi

    if [ -z "$cidr" ]; then
        printf -v cidr "10.42.%d.10/24" "$n"
    fi

    local lladdr
    printf -v lladdr "00:00:02:00:01:%02x" "$n"
    local lsp_name="${container}-${ns_name}-${if_name}"

    lxc_exec "$container" \
        "microovn.ovn-nbctl \
         -- \
         lsp-add $ls_name $lsp_name \
         -- \
         lsp-set-addresses $lsp_name '$lladdr $cidr'"
    lxc_exec "$container" \
        "microovn.ovs-vsctl \
         -- \
         add-port br-int $if_name \
         -- \
         set Interface $if_name type=internal external_ids:iface-id=$lsp_name"
    netns_ifadd "$container" "$ns_name" "$if_name" "$lladdr" "$cidr"
    wait_until "microovn_lsp_up $container $lsp_name"
}

# microovn_delete_vif CONTAINER NSNAME IFNAME
#
# Delete OVS interface with IFNAME and the LSP associated with it.
function microovn_delete_vif() {
    local container=$1; shift
    local ns_name=$1; shift
    local if_name=$1; shift

    local lsp_name="${container}-${ns_name}-${if_name}"

    lxc_exec "$container" "microovn.ovs-vsctl del-port $if_name"
    lxc_exec "$container" "microovn.ovn-nbctl lsp-del $lsp_name"
}

# ovn_chassis_registered CONTAINER CHASSIS
#
# Using ovn-sbctl on CONTAINER check that CHASSIS is registered
# in the Southbound database.
#
# Returns success if CHASSIS is registered, failure otherwise.
function ovn_chassis_registered() {
    local container=$1; shift
    local chassis=$1; shift

    run lxc_exec "$container" "microovn.ovn-sbctl show"
    grep -E "^Chassis $chassis\$" <<< "$output"
}

# setup_snap_aliases
function setup_snap_aliases(){
    local containers=$*
    for container in $containers; do
        for cmd in $(lxc_exec "$container" "ls /snap/bin/" |
                         grep -E 'microovn\.');
            do lxc_exec "$container" \
                "snap alias $(echo $cmd | sed 's/.*microovn\./microovn./') \
                            $(echo $cmd | sed 's/.*microovn.//')"
        done
    done
}

function install_ppa_netplan(){
    local containers=$*
    for container in $containers; do
        lxc_exec "$container" "add-apt-repository ppa:fnordahl/netplan-ovs-snap"
        lxc_exec "$container" "apt update"
        lxc_exec "$container" "apt install -y --allow-downgrades netplan.io=1.1.2-2~ubuntu24.04.1.0 libnetplan1=1.1.2-2~ubuntu24.04.1.0 netplan-generator=1.1.2-2~ubuntu24.04.1.0 python3-netplan=1.1.2-2~ubuntu24.04.1.0"
    done
}
