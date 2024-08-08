
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
    for container in $TEST_CONTAINERS; do
        # enable enabled service
        run lxc_exec "$container" "microovn enable switch"
        assert_output "Error: Failed to enable service 'switch': 'This Service is already enabled'"

        # enable non existing service
        run lxc_exec "$container" "microovn enable switchh"
        assert_output "Error: Failed to enable service 'switchh': 'Service does not exist'"

        # disable non existing service
        run lxc_exec "$container" "microovn disable switchh"
        assert_output "Error: Failed to disable service 'switchh': 'Service does not exist'"

        # disable service
        run lxc_exec "$container" "microovn disable switch"
        assert_output "Service switch disabled"

        # disable disabled service
        run lxc_exec "$container" "microovn disable switch"
        assert_output "Error: Failed to disable service 'switch': 'This service is not enabled'"

        run lxc_exec "$container" "microovn status | grep -ozE '${container}[^-]*' | grep switch"
        assert_output ""

        run lxc_exec "$container" "snap services microovn | grep switch | grep enabled"
        assert_output ""

        # enable disabled service
        run lxc_exec "$container" "microovn enable switch"
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
    assert_output "Error: Failed to disable service 'central': 'You cannot delete the final enabled central service'"

    # ensure central is actually still enabled
    assert [ -n "$(run lxc_exec "microovn-services-3" "microovn status | grep -ozE 'microovn-services-3[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "microovn-services-3" "snap services microovn | grep ovn-northd | grep enabled")"]
}


services_register_test_functions
