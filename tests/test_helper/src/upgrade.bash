function verify_that_currently_release_snap_can_be_upgraded() {
    echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV" >&3
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    perform_manual_upgrade_steps $TEST_CONTAINERS
}

# include: tls_cluster.bash