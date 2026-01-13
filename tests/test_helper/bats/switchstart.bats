
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

teardown() {
    print_diagnostics_on_failure $TEST_CONTAINERS
}

switchstart_register_test_functions() {
    bats_test_function \
        --description "Testing of starting switch before bootstrap" \
        -- start_switch_first_tests
}

start_switch_first_tests() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" "microovn status"
        assert_failure

        run lxc_exec "$container" "snap start microovn.switch"
        assert_success

        run lxc_exec "$container" "snap services microovn.switch |
                                   grep -q inactive"
        assert_failure

        run lxc_exec "$container" "microovn.ovs-vsctl show"
        assert_success
    done
}


switchstart_register_test_functions
