load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

load test_helper/bats/cluster.bats
load test_helper/bats/cli_ovsovn.bats
