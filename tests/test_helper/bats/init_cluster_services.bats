# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/tls.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS are populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]

    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
}

teardown() {
    local container
    for container in $TEST_CONTAINERS; do
        lxc_exec "$container" "snap remove --purge microovn"
        lxc file delete -q "$container/tmp/microovn.snap" >/dev/null 2>&1 || true
    done
}

init_cluster_services_register_test_functions() {
    bats_test_function \
        --description "Init MicroOVN in datapath mode (switch and chassis only)" \
        -- init_cluster_with_services "switch,chassis" "switch,chassis" "switch,chassis" "switch,chassis"

    bats_test_function \
        --description "Init MicroOVN in control mode (central only)" \
        -- init_cluster_with_services "central" "central" "central" "central"


    bats_test_function \
        --description "Init MicroOVN in 'auto' mode (default services)" \
        -- init_cluster_auto
}

# init_cluster_with_services SERVICE_LIST [ SERVICE_LIST ...]
#
# This test runs bootstraps the MicroOVN cluster via 'microovn init' interactive
# dialogue, enabling only services from SERVICE_LIST on each cluster member.
# SERVICE_LIST is a comma separated string with valid service names. It should
# be supplied as many times as there are nodes in TEST_CONTAINERS list.
init_cluster_with_services() {
    local services=$*

    local service_list
    read -r -a service_list <<< "$services"
    local service_list_len="${#service_list[@]}"

    local container_list
    read -r -a container_list <<< "$TEST_CONTAINERS"
    local container_list_len="${#container_list[@]}"

    # Ensure that services were specified for every test container
    assert_equal "$container_list_len" "$service_list_len"

    # Init individual cluster members with specified services enabled
    local leader
    local i
    for (( i=0; i<container_list_len; i++ )); do
        local container="${container_list[$i]}"
        local selected_services="${service_list[$i]}"
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$leader" ]; then
            microovn_init_create_cluster "$container" "$addr" "" "" "" "$selected_services"
            leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" "" "$selected_services"
        fi
    done

    declare -A expected_snap_services
    # Ensure that expected services are running on the node
    for (( i=0; i<container_list_len; i++ )); do
        local container="${container_list[$i]}"
        local selected_services="${service_list[$i]}"

        # Parse requested services into snap service names and the expected state
        run grep -w "switch" <<< "$selected_services"
        # shellcheck disable=SC2154 # Variable "$status" is exported from previous execution of 'run'
        expected_snap_services["switch"]=$([ "$status" -eq 0 ] && echo "enabled" || echo "disabled")

        run grep -w "chassis" <<< "$selected_services"
        expected_snap_services["chassis"]=$([ "$status" -eq 0 ] && echo "enabled" || echo "disabled")

        run grep -w "central" <<< "$selected_services"
        expected_snap_services["ovn-northd"]=$([ "$status" -eq 0 ] && echo "enabled" || echo "disabled")
        expected_snap_services["ovn-ovsdb-server-nb"]=$([ "$status" -eq 0 ] && echo "enabled" || echo "disabled")
        expected_snap_services["ovn-ovsdb-server-sb"]=$([ "$status" -eq 0 ] && echo "enabled" || echo "disabled")

        local service
        for service in "${!expected_snap_services[@]}"; do
            local expected_state="${expected_snap_services[$service]}"
            echo "# ($container) Checking that microovn.$service is $expected_state" >&3
            lxc_exec "$container" "snap services microovn.$service | grep $expected_state"
        done
    done
}

# init_cluster_auto
#
# This test bootstraps MicroOVN cluster via interactive 'init' dialogue,
# selecting "auto" as a value for requested services. This causes MicroOVn
# to fall back to the default behavior of enabling "chassis" and "switch" services
# on every node, and enabling "central" on the first three nodes.
init_cluster_auto() {
    # Init individual cluster members in "auto" mode
    local leader
    local container
    for container in $TEST_CONTAINERS; do
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$leader" ]; then
            microovn_init_create_cluster "$container" "$addr" "" "" "" "auto"
            leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" "" "auto"
        fi
    done

    local i=1
    for container in $TEST_CONTAINERS; do
        # Chassis and switch services are always enabled in "auto" mode
        echo "# ($container) Checking that microovn.switch is enabled" >&3
        lxc_exec "$container" "snap services microovn.switch | grep enabled"

        echo "# ($container) Checking that microovn.chassis is enabled" >&3
        lxc_exec "$container" "snap services microovn.chassis | grep enabled"

        # Central services should be enabled only on first three nodes in "auto" mode
        local expected_central_state
        expected_central_state=$([[ $i -le 3 ]] && echo "enabled" || echo "disabled")

        echo "# ($container) Checking that microovn.ovn-northd is $expected_central_state" >&3
        lxc_exec "$container" "snap services microovn.ovn-northd | grep $expected_central_state"

        echo "# ($container) Checking that microovn.ovn-ovsdb-server-nb is $expected_central_state" >&3
        lxc_exec "$container" "snap services microovn.ovn-ovsdb-server-nb | grep $expected_central_state"

        echo "# ($container) Checking that microovn.ovn-ovsdb-server-sb is $expected_central_state" >&3
        lxc_exec "$container" "snap services microovn.ovn-ovsdb-server-sb | grep $expected_central_state"
        ((++i))
    done
}

init_cluster_services_register_test_functions
