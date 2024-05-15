# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

teardown() {
    if [ "$BATS_TEST_DESCRIPTION" = "ovn-trace" ]; then
        for container in $TEST_CONTAINERS; do
            lxc_exec "$container" "microovn.ovn-nbctl lsp-del ovn-trace-p0"
            lxc_exec "$container" "microovn.ovn-nbctl ls-del ovn-trace"
            break
        done
    fi

    if [ "$BATS_TEST_DESCRIPTION" = "ovn-nbctl daemon" ]; then
        for container in $TEST_CONTAINERS; do
            lxc_exec "$container" "killall ovn-nbctl"
        done
    fi

    if [ "$BATS_TEST_DESCRIPTION" = "ovn-sbctl daemon" ]; then
        for container in $TEST_CONTAINERS; do
            lxc_exec "$container" "killall ovn-sbctl"
        done
    fi
}

@test "ovs-appctl ovs-vswitchd" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-appctl version"
        assert_success
        assert_output -p "ovs-vswitchd"
    done
}

@test "ovs-dpctl" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-dpctl dump-dps"
        assert_success
        assert_output "system@ovs-system"
    done
}

@test "ovs-ofctl" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-ofctl dump-flows br-int >/dev/null"
        assert_success
    done
}

@test "ovs-vsctl" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-vsctl show >/dev/null"
        assert_success
    done
}

@test "ovs-appctl ovsdb-server" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-appctl -t ovsdb-server version"
        assert_success
        assert_output -p "ovsdb-server"
    done
}

@test "ovn-appctl ovn-controller" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-appctl version"
        assert_success
        assert_output -p "ovn-controller"
    done
}

@test "ovn-appctl ovn-northd" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-appctl -t ovn-northd version"
        assert_success
        assert_output -p "ovn-northd"
    done
}

@test "ovn-nbctl" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-nbctl show > /dev/null"
        assert_success
    done
}

@test "ovn-nbctl daemon" {
    local ovn_nbctl_socket

    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "OVN_NB_DAEMON=/dev/null microovn.ovn-nbctl show"
        assert_failure

        ovn_nbctl_socket=$(lxc_exec "$container" \
            "microovn.ovn-nbctl --detach")
        run lxc_exec "$container" \
            "OVN_NB_DAEMON=$ovn_nbctl_socket microovn.ovn-nbctl show"
        assert_success
    done
}

@test "ovn-sbctl" {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-sbctl show > /dev/null"
        assert_success
    done
}

@test "ovn-sbctl daemon" {
    local ovn_sbctl_socket

    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "OVN_SB_DAEMON=/dev/null microovn.ovn-sbctl show"
        assert_failure

        ovn_sbctl_socket=$(lxc_exec "$container" \
            "microovn.ovn-sbctl --detach")
        run lxc_exec "$container" \
            "OVN_SB_DAEMON=$ovn_sbctl_socket microovn.ovn-sbctl show"
        assert_success
    done
}

@test "ovn-trace" {
    local first_container
    for container in $TEST_CONTAINERS; do
        if [ -z "$first_container" ]; then
            first_container=$container
            run lxc_exec "$first_container" \
                "microovn.ovn-nbctl ls-add ovn-trace"
            assert_success
            run lxc_exec "$first_container" \
                "microovn.ovn-nbctl --wait=hv lsp-add ovn-trace ovn-trace-p0"
            assert_success
        fi
        run lxc_exec "$container" \
            "microovn.ovn-trace --ovs ovn-trace \
                 'inport==\"ovn-trace-p0\" && \
                 eth.type == 0x800'"
        assert_success
        assert_output -p 'ingress(dp="ovn-trace", inport="ovn-trace-p0")'
        refute_output -p ERR
        refute_output -p WARN
    done
}
