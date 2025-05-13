
load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

services_register_test_functions() {
    bats_test_function \
        --description "Testing of service functionality" \
        -- service_tests
    bats_test_function \
        --description "Testing of service control warnings and errors" \
        -- service_warning_tests
}

service_tests() {
    read -r -a containers <<< "$TEST_CONTAINERS"
    for container in $TEST_CONTAINERS; do
        # enable enabled service
        run lxc_exec "$container" "microovn enable switch"
        assert_output "Error: failed to enable service 'switch': 'this service is already enabled'"

        # enable non existing service
        run lxc_exec "$container" "microovn enable switchh"
        assert_output "Error: invalid argument \"switchh\" for \"microovn enable\""

        # disable service
        run lxc_exec "$container" "microovn disable switch"
        assert_output "Service switch disabled"

        # disable disabled service
        run lxc_exec "$container" "microovn disable switch"
        assert_output "Error: failed to disable service 'switch': 'this service is not enabled'"

        run lxc_exec "$container" "microovn status | grep -ozE '${container}[^-]*' | grep switch"
        assert_output ""

        run lxc_exec "$container" "snap services microovn | grep switch | grep enabled"
        assert_output ""

        # enable disabled service
        run lxc_exec "$container" "microovn enable switch"
        assert_output "Service switch enabled"

        assert [ -n "$(run lxc_exec "$container" "microovn status | grep -ozE '${container}[^-]*' | grep switch")"]
        assert [ -n "$(run lxc_exec "$container" "snap services microovn | grep switch | grep enabled")"]

        # disable service remotely with nonexisting container
        run lxc_exec "$container" "microovn disable switch --node vriskaserket"
        assert_failure
        assert_output "Error: failed to disable service 'switch': 'Failed to get cluster member for request target name \"vriskaserket\": CoreClusterMember not found'"

        # enable service remotely with nonexisting container
        run lxc_exec "$container" "microovn enable switch --node vriskaserket"
        assert_failure
        assert_output "Error: failed to enable service 'switch': 'Failed to get cluster member for request target name \"vriskaserket\": CoreClusterMember not found'"

        # disable service remotely
        run lxc_exec "${containers[0]}" "microovn disable switch --node ${container}"
        assert_output "Service switch disabled"

        run lxc_exec "$container" "microovn status | grep -ozE '${container}[^-]*' | grep switch"
        assert_output ""

        run lxc_exec "$container" "snap services microovn | grep switch | grep enabled"
        assert_output ""

        # enable disabled service remotely
        run lxc_exec "${containers[0]}" "microovn enable switch --node ${container}"
        assert_output "Service switch enabled"


        assert [ -n "$(run lxc_exec "$container" "microovn status | grep -ozE '${container}[^-]*' | grep switch")"]
        assert [ -n "$(run lxc_exec "$container" "snap services microovn | grep switch | grep enabled")"]
    done
}

service_warning_tests() {
    run lxc_exec "microovn-services-1" "microovn disable central -v 1>/dev/null"
    assert_output -p "[central] Warning: Cluster with even number of members has same fault tolerance, but higher quorum requirements, than cluster with one less member."
    assert_output -p "[central] Warning: Cluster with less than 3 nodes can't tolerate any node failures."

    run lxc_exec "microovn-services-2" "microovn disable central -v 1>/dev/null"
    assert_output -p "[central] Warning: Cluster with less than 3 nodes can't tolerate any node failures."

    run lxc_exec "microovn-services-3" "microovn disable central"
    assert_output "Error: failed to disable service 'central': 'cannot disable last central node without explicit confirmation'"

    # ensure central is actually still enabled
    assert [ -n "$(run lxc_exec "microovn-services-3" "microovn status | grep -ozE 'microovn-services-3[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "microovn-services-3" "snap services microovn | grep ovn-northd | grep enabled")"]

    # Explicitly allow disabling last central node
    run lxc_exec "microovn-services-3" "microovn disable central --allow-disable-last-central"

    # ensure central was disabled
    assert [ -z "$(run lxc_exec "microovn-services-3" "microovn status | grep -ozE 'microovn-services-3[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "microovn-services-3" "snap services microovn | grep ovn-northd | grep disabled")"]
}


services_register_test_functions
