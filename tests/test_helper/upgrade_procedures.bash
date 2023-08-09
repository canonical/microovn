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
}

# perform_manual_upgrade_steps CONTAINER1 [CONTAINER2 ...]
#
# Sequentially execute manual steps that are required for upgrade
# between certain MicroOVN snap revisions.
function perform_manual_upgrade_steps() {
    local containers=$*; shift

    if [ "$MICROOVN_SNAP_REV" -lt 111 ]; then
        revision_111_upgrade_tls "$containers"
    fi
}
