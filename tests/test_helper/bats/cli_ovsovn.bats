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

cli_ovsovn_register_test_functions() {
    bats_test_function \
        --description "ovs-appctl ovs-vswitchd" \
        -- ovs-appctl_ovs-vswitchd
    bats_test_function \
        --description "ovs-dpctl" \
        -- ovs-dpctl
    bats_test_function \
        --description "ovs-ofctl" \
        -- ovs-ofctl
    bats_test_function \
        --description "ovs-vsctl" \
        -- ovs-vsctl
    bats_test_function \
        --description "ovs-appctl ovsdb-server" \
        -- ovs-appctl_ovsdb-server
    bats_test_function \
        --description "ovn-appctl ovn-controller" \
        -- ovn-appctl_ovn-controller
    bats_test_function \
        --description "ovn-appctl ovn-northd" \
        -- ovn-appctl_ovn-northd
    bats_test_function \
        --description "ovn-nbctl" \
        -- ovn-nbctl
    bats_test_function \
        --description "ovn-nbctl daemon" \
        -- ovn-nbctl_daemon
    bats_test_function \
        --description "ovn-sbctl" \
        -- ovn-sbctl
    bats_test_function \
        --description "ovn-sbctl daemon" \
        -- ovn-sbctl_daemon
    bats_test_function \
        --description "ovn-trace" \
        -- ovn-trace
    bats_test_function \
        --description "microovn --version" \
        -- microovn_version
    bats_test_function \
        --description "Test waitready command"\
        -- test_waitready
    bats_test_function \
        --description "Test enable disable microovn"\
        -- test_disable_microovn
}

ovs-appctl_ovs-vswitchd() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-appctl version"
        assert_success
        assert_output -p "ovs-vswitchd"
    done
}

ovs-dpctl() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-dpctl dump-dps"
        assert_success
        assert_output "system@ovs-system"
    done
}

ovs-ofctl() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-ofctl dump-flows br-int >/dev/null"
        assert_success
    done
}

ovs-vsctl() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-vsctl show >/dev/null"
        assert_success
    done
}

ovs-appctl_ovsdb-server() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovs-appctl -t ovsdb-server version"
        assert_success
        assert_output -p "ovsdb-server"
    done
}

ovn-appctl_ovn-controller() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-appctl version"
        assert_success
        assert_output -p "ovn-controller"
    done
}

ovn-appctl_ovn-northd() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-appctl -t ovn-northd version"
        assert_success
        assert_output -p "ovn-northd"
    done
}

ovn-nbctl() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-nbctl show > /dev/null"
        assert_success
    done
}

ovn-nbctl_daemon() {
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

ovn-sbctl() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" \
            "microovn.ovn-sbctl show > /dev/null"
        assert_success
    done
}

ovn-sbctl_daemon() {
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

ovn-trace() {
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

microovn_version() {
    for container in $TEST_CONTAINERS; do
        run lxc_exec "$container" "microovn --version"
        assert_success
        assert_line --index 0 --regexp '^microovn: [^[:space:]]+'
        assert_line --index 1 --regexp '^ovn: [^[:space:]]+'
        assert_line --index 2 --regexp '^openvswitch: [^[:space:]]+'

        # only need to check this on first container.
        break
    done
}

cli_ovsovn_register_test_functions

test_waitready(){
    for container in $TEST_CONTAINERS; do
        run lxc_exec $container "snap stop microovn.daemon"
        assert_success
        run lxc_exec $container "microovn waitready -t 1"
        assert_failure
        run lxc_exec $container "snap start microovn.daemon"
        assert_success
        run lxc_exec $container "microovn waitready -t 10"
        assert_success
    done;
}

test_disable_microovn(){
    for container in $TEST_CONTAINERS; do
        run lxc_exec $container "snap disable microovn"
        assert_success
        run lxc_exec $container "snap enable microovn"
        assert_success
        sleep 5
        run lxc_exec $container "microovn status"
        assert_success
        run lxc_exec $container "microovn.ovs-vsctl show"
        assert_success
        run lxc_exec $container "microovn.ovn-nbctl --wait=sb lr-add test"
        assert_success
        run lxc_exec $container "microovn.ovn-nbctl --wait=sb lr-del test"
        assert_success
    done;
}

cli_ovsovn_register_test_functions
