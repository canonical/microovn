# This is a bash shell fragment -*- bash -*-

setup_file() {
    ABS_TOP_TEST_DIRNAME="${BATS_TEST_DIRNAME}/"
    export ABS_TOP_TEST_DIRNAME
}

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash
}

teardown() {
    delete_containers $TEST_CONTAINERS
}

@test "Scale cluster from 1 through 4 nodes" {
    local final_status=0
    local max_containers=4

    readarray -d " " -t containers < <(container_names \
                                       "$BATS_TEST_FILENAME" \
                                       "$max_containers")
    declare -A starttimes_ovn_controller

    for (( i=1; i<=max_containers; i++ )) {
        local container
        container=${containers[*]:$((( $i - 1 ))):1}

        launch_containers $container
        wait_containers_ready $container
        install_microovn "$MICROOVN_SNAP_PATH" $container

        local addr
        addr=$(container_get_default_ip \
                  $container \
                  "$(test_is_ipv6_test && echo inet6 || echo inet)")
        if [ $i -eq 1 ]; then
            microovn_init_create_cluster "$container" "$addr" ""
        else
            local token
            token=$(microovn_cluster_get_join_token \
                        "${containers[0]}" \
                        "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" ""
        fi
        starttimes_ovn_controller[$container]=$(\
            microovn_wait_for_service_starttime $container ovn-controller)
        echo "starttime ${container} ovn-controller: \
            ${starttimes_ovn_controller[$container]}"

        TEST_CONTAINERS="${containers[*]:0:$i}"
        export TEST_CONTAINERS

        local test_filename=${ABS_TOP_TEST_DIRNAME}
        test_filename+=test_helper/bats/scaleup_cluster
        test_filename+=$(test_is_ipv6_test && echo _ipv6 || true)
        test_filename+=.bats

        if [ $i -gt 1 ]; then
            echo "# Rerunning tests after scaling to $i containers" >&3
        fi

        # Note that the outer bats runner will perform validation on the
        # number of tests ran based on TAP output, so it is important that
        # the inner bats runner uses a different format for its output.
        run bats -F junit $test_filename

        echo "# $output" >&3
        echo "#" >&3

        final_status=$(($final_status + $status))
    }

    [ "$final_status" -eq 0 ]

    echo "# Ensure ovn-controller was not restarted for scaling events"
    for (( i=0; i<max_containers; i++ )) {
        local container
        container=${containers[*]:$i:1}

        starttime_ovn_controller=$(\
            microovn_wait_for_service_starttime $container ovn-controller)
        starttime_ovn_controller=$(get_pid_start_time $container \
            "$(microovn_get_service_pid $container ovn-controller)")
        echo "starttime ${container} ovn-controller: $starttime_ovn_controller"
        assert [ $starttime_ovn_controller -eq \
                 ${starttimes_ovn_controller[$container]} ]
    }
}
