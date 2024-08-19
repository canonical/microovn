# rejoin_cluster_with_tls CONTAINER DB_TYPE TARGET_IP TARGET_PROTO MONITOR1 [MONITOR2 ...]
#
# Rejoin OVN OVSDB cluster on with member running on CONTAINER using tls.
#
# This function requires DB_TYPE which must be either 'nb' or 'sb'. It then disconnects CONTAINER
# from cluster and rejoin the cluster again using TARGET_IP and TARGET_PROTO (either 'tcp' or 'ssl').
#
# MONITORs should be list of remaining containers in the cluster that are used to verify that CONTAINER
# left the cluster before attempting to rejoin it.
function rejoin_cluster_with_tls() {
    local target_container=$1; shift
    local db_type=$1; shift
    local target_ip=$1; shift
    local target_proto=$1; shift
    local monitor_containers=$*
    local ctl_path="/var/snap/microovn/common/run/ovn/"
    local db_path="/var/snap/microovn/common/data/central/db/"
    local local_ip=""
    local_ip=$(microovn_get_cluster_address "$target_container")
    local target_server_id=""
    target_server_id=$(microovn_ovndb_server_id "$target_container" "$db_type")

    if [ "$db_type" == "nb" ]; then
        local port="6643"
        local db_name="OVN_Northbound"
        ctl_path="$ctl_path/ovnnb_db.ctl"
        db_path="$db_path/ovnnb_db.db"

        elif [ "$db_type" == "sb" ]; then
            local port="6644"
            local db_name="OVN_Southbound"
            ctl_path="$ctl_path/ovnsb_db.ctl"
            db_path="$db_path/ovnsb_db.db"

        else
            echo "# Unknown database type '$db_type'. Valid values: 'nb', 'sb'"
            return 1
    fi
    echo "# ($target_container) Rejoining $db_name cluster with TLS config. \
          Target: $target_proto:$target_ip:$port"


    lxc_exec "$target_container" "microovn.ovn-appctl -t $ctl_path cluster/leave $db_name"
    wait_ovsdb_cluster_container_leave "$target_server_id" "$ctl_path" "$db_name" 30 "$monitor_containers"
    lxc_exec "$target_container" "snap stop microovn.ovn-northd"
    lxc_exec "$target_container" "snap stop microovn.ovn-ovsdb-server-nb"
    lxc_exec "$target_container" "snap stop microovn.ovn-ovsdb-server-sb"
    lxc_exec "$target_container" "rm $db_path"
    lxc_exec "$target_container" "microovn.ovsdb-tool join-cluster $db_path $db_name \
                                  ssl:$local_ip:$port $target_proto:$target_ip:$port"
    lxc_exec "$target_container" "snap restart microovn.ovn-ovsdb-server-nb"
    lxc_exec "$target_container" "snap restart microovn.ovn-ovsdb-server-sb"
    lxc_exec "$target_container" "snap restart microovn.ovn-northd"
    wait_ovsdb_cluster_changes_applied "$target_container" "$ctl_path" "$db_name" 30
}

# ovsdb_rebuild_tls_cluster DB_TYPE CONTAINER1 [CONTAINER2 ...]
#
# Gradually rebuild OVN OVSDB cluster from tcp connection to tls connection.
#
# This function preserves overall cluster data by gradually removing cluster members that use tcp
# connection and rejoining them using tls. It preserves overall cluster integrity by removing only
# one container at a time before rejoining, ensuring that cluster is in converged state and then moving
# on to another cluster member.
#
# Parameter DB_TYPE identifies which database cluster should be rebuilt. Accepted values are 'nb' or 'sb'
function ovsdb_rebuild_tls_cluster() {
    local db_type=$1; shift
    local containers=$*
    local target_proto="tcp"
    local target_ip=""
    local last_container=""
    last_container=$(awk '{print $NF;}' <<< "$containers")
    target_ip=$(microovn_get_cluster_address "$last_container")
    local first_run=1

    for container in $containers; do
        local monitor_containers=""
        monitor_containers=$(sed "s/\<$container\>//" <<< "$containers")
        rejoin_cluster_with_tls "$container" "$db_type" "$target_ip" $target_proto "$monitor_containers"
        if [ $first_run -eq 1 ]; then
            target_proto="ssl"
            target_ip=$(microovn_get_cluster_address "$container")
            first_run=0
        fi
    done
}

# revision_111_upgrade_tls CONTAINER1 [CONTAINER2 ...]
#
# Perform manual steps needed when upgrading from snap revision lower than
# 111. This upgrade enables TLS communication between OVN cluster members
# and includes these manual steps:
#   * Ensure that all MicroOVN members are ONLINE
#   * run 'microovn certificates regenerate-ca' on one of the members
#   * Restart 'microovn.daemon' on all members
function revision_111_upgrade_tls() {
    local containers=$*
    local upgrade_leader=""
    upgrade_leader=$(echo "$containers" | awk '{print $1;}')
    assert [ -n "$upgrade_leader" ]

    wait_microovn_online "$upgrade_leader" 30
    lxc_exec "$upgrade_leader" "microovn certificates regenerate-ca"
    for container in $containers ; do
        lxc_exec "$container" "snap restart microovn.daemon"
    done
    wait_microovn_online "$upgrade_leader" 30

    local central_containers=""
    for container in $containers; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"central"* ]]; then
            central_containers+="$container "
        fi
    done

    echo "# Rebuilding for OVN Central containers using tls: $central_containers"

    ovsdb_rebuild_tls_cluster "nb" "$central_containers"
    ovsdb_rebuild_tls_cluster "sb" "$central_containers"
}

# maybe_perform_manual_upgrade_steps CONTAINER1 [CONTAINER2 ...]
#
# Sequentially execute manual steps that are required for upgrade
# between certain MicroOVN snap revisions.
function maybe_perform_manual_upgrade_steps() {
    local containers=$*; shift

    if [ "$MICROOVN_SNAP_REV" -lt 111 ]; then
        revision_111_upgrade_tls "$containers"
    fi
}
