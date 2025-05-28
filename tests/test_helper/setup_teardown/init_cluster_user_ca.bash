setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/tls.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    CENTRAL_CONTAINERS=""
    CHASSIS_CONTAINERS=""

    export TEST_CONTAINERS
    export CENTRAL_CONTAINERS
    export CHASSIS_CONTAINERS

    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    export USER_CA_CRT="/var/snap/microovn/common/ca.crt"
    export USER_CA_KEY="/var/snap/microovn/common/ca.key"
    export LEADER
    for container in $TEST_CONTAINERS; do
        local addr
        addr=$(container_get_default_ip "$container" "inet")
        assert [ -n "$addr" ]
        if [ -z "$LEADER" ]; then
            generate_user_ca "$container" "ec" "$USER_CA_CRT" "$USER_CA_KEY"
            microovn_init_create_cluster "$container" "$addr" "" "$USER_CA_CRT" "$USER_CA_KEY"
            LEADER="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$LEADER" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" ""
        fi
    done

    # Categorize containers as "CENTRAL" and "CHASSIS" based on the services they run
    for container in $TEST_CONTAINERS; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"central"* ]]; then
            CENTRAL_CONTAINERS+="$container "
        else
            CHASSIS_CONTAINERS+="$container "
        fi
    done
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}

