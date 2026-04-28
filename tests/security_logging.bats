# This is a bash shell fragment -*- bash -*-

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

# --- Test functions ---

# Verify that security log events are emitted by default after bootstrap.
# The daemon startup produces a "sys_startup" security event that must
# appear in the journal.
security_logging_enabled_by_default() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    echo "# Checking that security log events are present by default" >&3

    # The daemon has already started during bootstrap. Its journal should
    # contain at least the sys_startup security event.
    run lxc_exec "$container" \
        "journalctl -u snap.microovn.daemon --no-pager | grep 'security=true'"
    assert_success

    run lxc_exec "$container" \
        "journalctl -u snap.microovn.daemon --no-pager | grep 'event=sys_startup'"
    assert_success
}

# Verify that setting security-logging=false suppresses security events.
# After restarting the daemon with the option disabled, no new security
# log entries should appear.
security_logging_disabled_with_flag() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    echo "# Disabling security logging via snap set" >&3
    run lxc_exec "$container" "snap set microovn security-logging=false"
    assert_success

    # Record a journal cursor so we only inspect entries produced after
    # the restart.
    local cursor
    run lxc_exec "$container" \
        "journalctl -u snap.microovn.daemon --no-pager -n 0 --show-cursor \
         | grep '^-- cursor:' | awk '{print \$3}'"
    assert_success
    # shellcheck disable=SC2154 # Variable "$output" is exported from previous execution of 'run'
    cursor="$output"
    assert [ -n "$cursor" ]

    echo "# Restarting daemon with security logging disabled" >&3
    run lxc_exec "$container" "snap restart microovn.daemon"
    assert_success

    # Give the daemon a moment to complete its startup sequence
    sleep 5

    # Verify the daemon is running (the restart succeeded)
    run lxc_exec "$container" "snap services microovn.daemon | grep active"
    assert_success

    # There should be no security=true entries after the cursor
    run lxc_exec "$container" \
        "journalctl -u snap.microovn.daemon --no-pager --after-cursor='${cursor}' \
         | grep 'security=true' || true"
    assert_output ""
}

# Verify that the configure hook rejects invalid values for the
# security-logging snap option.
security_logging_rejects_invalid_value() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    echo "# Setting security-logging to an invalid value" >&3
    run lxc_exec "$container" "snap set microovn security-logging=invalid"
    assert_failure

    # Verify the error message mentions the invalid value
    assert_output -p 'security-logging must be "true" or "false"'
}

# --- Register test functions ---

security_logging_register_test_functions() {
    bats_test_function \
        --description "Security log events are emitted by default" \
        -- security_logging_enabled_by_default
    bats_test_function \
        --description "Security logging stops when disabled via snap set" \
        -- security_logging_disabled_with_flag
    bats_test_function \
        --description "Invalid security-logging value is rejected" \
        -- security_logging_rejects_invalid_value
}

security_logging_register_test_functions
