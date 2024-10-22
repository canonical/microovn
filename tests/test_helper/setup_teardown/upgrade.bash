setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/upgrade_procedures.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    ABS_TOP_TEST_DIRNAME="${BATS_TEST_DIRNAME}/"
    export ABS_TOP_TEST_DIRNAME

    # Determine MicroOVN channel from which we are upgrading
    test_name=$(basename "$BATS_TEST_FILENAME")
    MICROOVN_SNAP_CHANNEL=$(get_upgrade_test_version "$test_name" "$TEST_NAME_PREFIX")

    # Create test deployment
    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    CENTRAL_CONTAINERS=""
    CHASSIS_CONTAINERS=""

    export TEST_CONTAINERS
    export CENTRAL_CONTAINERS
    export CHASSIS_CONTAINERS

    launch_containers_args \
        "${TEST_LXD_LAUNCH_ARGS:--c security.nesting=true}" $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn_from_store "$MICROOVN_SNAP_CHANNEL" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS

    # Categorize containers as "CENTRAL" and "CHASSIS" based on the services they run
    local container=""
    for container in $TEST_CONTAINERS; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"central"* ]]; then
            CENTRAL_CONTAINERS+="$container "
        else
            CHASSIS_CONTAINERS+="$container "
        fi
    done

    # Make sure that microcluster is fully converged before proceeding.
    # Performing further actions before the microcluster is ready may lead to
    # unexpectedly long convergence after a microcluster schema upgrade.
    for container in $TEST_CONTAINERS; do
        wait_microovn_online "$container" 60
    done

    # detect and export initial MicroOVN snap revision
    container=$(echo "$TEST_CONTAINERS" | awk '{print $1;}' )
    export MICROOVN_SNAP_REV=""
    MICROOVN_SNAP_REV=$(lxc_exec "$container" "snap list | grep microovn | awk '{print \$3;}'")
    assert [ -n "$MICROOVN_SNAP_REV" ]

    if [ -n "$UPGRADE_DO_UPGRADE" ]; then
        assert [ -n "$CENTRAL_CONTAINERS" ]
        assert [ -n "$CHASSIS_CONTAINERS" ]

        # Export names used locally on chassis containers for use in
        # teardown_file().
        export UPGRADE_NS_NAME="upgrade_ns0"
        export UPGRADE_VIF_NAME="upgrade_vif0"

        # Set up gateway router, workload and background ping on each chassis.
        for container in $CHASSIS_CONTAINERS; do
            local ctn_n
            ctn_n=$(microovn_extract_ctn_n "$container")
            microovn_add_gw_router "$container"
            netns_add "$container" "$UPGRADE_NS_NAME"
            microovn_add_vif "$container" \
                "$UPGRADE_NS_NAME" "$UPGRADE_VIF_NAME"
            ping_start "$container" 10.42.${ctn_n}.1 "$UPGRADE_NS_NAME"
        done

        echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV "\
             "on central container(s)." >&3
        install_microovn "$MICROOVN_SNAP_PATH" $CENTRAL_CONTAINERS

        for container in $CENTRAL_CONTAINERS; do
            microovn_wait_ovndb_state "$container" nb connected 32
            microovn_wait_ovndb_state "$container" sb connected 32
        done

        maybe_perform_manual_upgrade_steps $CENTRAL_CONTAINERS

        # Reap ping and assert on result.
        #
        # Start background ping for next measurement.
        for container in $CHASSIS_CONTAINERS; do
            local ctn_n
            ctn_n=$(microovn_extract_ctn_n "$container")
            local n_lost
            n_lost=$(ping_packets_lost \
                "$container" 10.42.${ctn_n}.1 "$UPGRADE_NS_NAME")
            # Apart from the one packet that can be lost while stopping
            # ``ping``, we expect zero loss.
            assert test "$n_lost" -le 1

            ping_start "$container" 10.42.${ctn_n}.1 "$UPGRADE_NS_NAME"
        done

        echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV "\
             "on chassis container(s)." >&3
        install_microovn "$MICROOVN_SNAP_PATH" $CHASSIS_CONTAINERS

        # Now that the remaining containers have been upgraded any pending
        # schema conversions will be performed both for the microcluster and
        # OVSDB databases.  Ensure these processes are complete before
        # measuring the result.
        for container in $TEST_CONTAINERS; do
            wait_microovn_online "$container" 60
            for db in nb sb; do
                local cmd
                printf -v cmd \
                    'microovn_status_is_schema_ok %s %s' "$container" "$db"
                wait_until "$cmd"
            done
        done

        # Reap ping and assert on result.
        for container in $CHASSIS_CONTAINERS; do
            local ctn_n
            ctn_n=$(microovn_extract_ctn_n "$container")
            local max_lost=8
            local n_lost
            n_lost=$(ping_packets_lost \
                "$container" 10.42.${ctn_n}.1 "$UPGRADE_NS_NAME")
            # Upgrading the node with the instance being monitored will
            # inevitably cause some data path interruption as Open vSwitch
            # restarts.
            assert test "$n_lost" -le "$max_lost"
            echo "# Upgrade induced packet loss: $n_lost packets " \
                 "(threshold $max_lost)" >&3
        done
    fi
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS

    if [ -n "$UPGRADE_NS_NAME" ] && [ -n "$UPGRADE_VIF_NAME" ]; then
        local container
        for container in $CHASSIS_CONTAINERS; do
            microovn_delete_vif "$container" \
                "$UPGRADE_NS_NAME" "$UPGRADE_VIF_NAME"
            netns_delete "$container" "$UPGRADE_NS_NAME"
            microovn_delete_gw_router "$container"
        done
    fi

    delete_containers $TEST_CONTAINERS
}
