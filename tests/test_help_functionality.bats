# This is a bash shell fragment -*- bash -*-

# Load the required helper scripts
load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated to avoid false positives
    assert [ -n "$TEST_CONTAINERS" ]
}

teardown() {
    # No specific cleanup needed for this test
    :
}

# Register the new test functions
cli_help_functionality_register_test_functions() {
    bats_test_function \
        --description "Test invalid arguments return error code 1" \
        -- test_invalid_args_return_1

    bats_test_function \
        --description "Test valid arguments return code 0" \
        -- test_valid_args_return_0
}

test_invalid_args_return_1() {
    for container in $TEST_CONTAINERS; do
        # Run the cluster add command with invalid arguments
        run lxc_exec "$container" "microovn cluster add"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the cluster remove command with invalid arguments
        run lxc_exec "$container" "microovn cluster remove"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the cluster bootstrap command with invalid arguments
        run lxc_exec "$container" "microovn cluster bootstrap invalid_arg"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the cluster join command with invalid arguments
        run lxc_exec "$container" "microovn cluster join"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the certificate reissue command with invalid arguments
        run lxc_exec "$container" "microovn certificates reissue"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the enable command with invalid arguments
        run lxc_exec "$container" "microovn enable"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

        # Run the cluster disable command with invalid arguments
        run lxc_exec "$container" "microovn disable"

        # Assert the return code is 1
        assert_failure
        assert [ "$status" -eq 1 ]

    done
}

test_valid_args_return_0() {
    for container in $TEST_CONTAINERS; do
        # Run the command with valid arguments
        run lxc_exec "$container" "microovn --version"

        # Assert the return code is 0
        assert_success
        assert [ "$status" -eq 0 ]

        # Ensure help message is not in the output
        refute_output --partial "Usage: microovn"
    done
}

# Register the new test functions
cli_help_functionality_register_test_functions
