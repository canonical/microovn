# This is a bash shell fragment -*- bash -*-

# Define test filename prefix that helps to determine from which version should
# the upgrade be tested.
export TEST_NAME_PREFIX="upgrade"

# Instruct the ``setup_file`` function to perform the upgrade.
export UPGRADE_DO_UPGRADE=1
load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

# Load test cases to run after the upgrade is complete.  Note that only files
# that dynamically define the tests with ``bats_test_function`` can be loaded.
#
# Skip auto-renew tests as they require VM and are not relevant to the upgrade process
export SKIP_TLS_RENEW=1
load test_helper/bats/cluster.bats
load test_helper/bats/tls_cluster.bats
