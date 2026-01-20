# This is a bash shell fragment -*- bash -*-

# Define test filename prefix that helps to determine from which version should
# the upgrade be tested.
export TEST_NAME_PREFIX="single_node_ovsdb_schema_upgrade"

export TEST_N_CONTAINERS=1

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

load test_helper/bats/ovsdb_schema_upgrade.bats
