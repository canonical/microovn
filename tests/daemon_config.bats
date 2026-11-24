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

# Verify that `microovn daemon-config list` reports the daemon configuration
# keys exposed over the /1.0/daemon/config endpoint.
daemon_config_lists_available_keys() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config list"
    assert_success
    assert_output -p "name"
    assert_output -p "address"
    assert_output -p "servers"
    assert_output -p "failure-domain"
}

# Verify that an unset failure-domain reports 0 (MicroCluster's default).
daemon_config_gets_default_failure_domain() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config get failure-domain"
    assert_success
    assert_output "0"
}

# Verify that the read-only daemon fields (name/address) are retrievable.
daemon_config_gets_immutable_fields() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config get name"
    assert_success
    assert_output "$container"

    run lxc_exec "$container" "microovn daemon-config get address"
    assert_success
    assert_output -p ":6686"
}

# Verify that setting the failure-domain is persisted to daemon.yaml and is
# read back. Setting triggers a local dqlite roll (leader handoff) so the value
# takes effect without a full daemon restart.
daemon_config_sets_and_gets_failure_domain() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config set failure-domain 7"
    assert_success

    run lxc_exec "$container" "microovn daemon-config get failure-domain"
    assert_success
    assert_output "7"

    run lxc_exec "$container" \
        "cat /var/snap/microovn/common/state/daemon.yaml"
    assert_success
    assert_output -p "failure-domain: 7"

    # The dqlite roll must not have taken down the daemon.
    run lxc_exec "$container" "snap services microovn.daemon | grep active"
    assert_success
}

# Verify that an already-set failure-domain can be updated.
daemon_config_updates_failure_domain() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config set failure-domain 42"
    assert_success

    run lxc_exec "$container" "microovn daemon-config get failure-domain"
    assert_success
    assert_output "42"
}

# Verify that the failure-domain can be cleared back to the default (0).
daemon_config_clears_failure_domain() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config set failure-domain 0"
    assert_success

    run lxc_exec "$container" "microovn daemon-config get failure-domain"
    assert_success
    assert_output "0"
}

# Verify that unknown daemon configuration keys are rejected.
daemon_config_rejects_unknown_key() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config get bogus"
    assert_failure
    assert_output -p "not recognized"

    run lxc_exec "$container" "microovn daemon-config set bogus 1"
    assert_failure
    assert_output -p "not settable"
}

# Verify that read-only daemon fields cannot be written through `set`.
daemon_config_rejects_non_settable_key() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config set name attacker"
    assert_failure
    assert_output -p "not settable"

    run lxc_exec "$container" "microovn daemon-config set servers attacker"
    assert_failure
    assert_output -p "not settable"
}

# Verify that non-integer or out-of-range failure-domain values are rejected.
daemon_config_rejects_invalid_value() {
    local container
    read -r container <<< "$TEST_CONTAINERS"

    run lxc_exec "$container" "microovn daemon-config set failure-domain not-a-number"
    assert_failure
    assert_output -p "must be an unsigned 64-bit integer"

    run lxc_exec "$container" "microovn daemon-config set failure-domain 18446744073709551616"
    assert_failure
    assert_output -p "must be an unsigned 64-bit integer"
}

# --- Register test functions ---

daemon_config_register_test_functions() {
    bats_test_function \
        --description "daemon-config list exposes daemon configuration keys" \
        -- daemon_config_lists_available_keys
    bats_test_function \
        --description "failure-domain defaults to 0 when unset" \
        -- daemon_config_gets_default_failure_domain
    bats_test_function \
        --description "daemon-config get returns name and address" \
        -- daemon_config_gets_immutable_fields
    bats_test_function \
        --description "daemon-config sets and persists the failure-domain" \
        -- daemon_config_sets_and_gets_failure_domain
    bats_test_function \
        --description "failure-domain can be updated after being set" \
        -- daemon_config_updates_failure_domain
    bats_test_function \
        --description "failure-domain can be cleared back to 0" \
        -- daemon_config_clears_failure_domain
    bats_test_function \
        --description "unknown daemon configuration keys are rejected" \
        -- daemon_config_rejects_unknown_key
    bats_test_function \
        --description "read-only daemon fields cannot be written" \
        -- daemon_config_rejects_non_settable_key
    bats_test_function \
        --description "invalid failure-domain values are rejected" \
        -- daemon_config_rejects_invalid_value
}

daemon_config_register_test_functions
