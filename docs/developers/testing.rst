.. _run MicroOVN tests:

==================
Run MicroOVN tests
==================

MicroOVN has two types of tests, linter checks and functional tests and this
page will show how to run them.

Linter checks
-------------

Go code
~~~~~~~

We make use of `golangci-lint`_ and you can find a list of enabled linters in
the ``microovn/.golangci.yml`` configuration file.

Successfully running the tool requires build dependencies to be installed and
build environment variables properly set up.

Developer ergonomics are important to us, and we want the same experience in
local development environments as in our gate.

As such we have opted to run `golangci-lint`_ as part of the snap build
process as it gives us consistent results in both environments and relieves
the developer of the burden of manually installing build dependencies to
perform the checks locally.

If you use an IDE with `golangci-lint`_ support and want to utilise it, the
tool should automatically discover this configuration.  You will however need
to install additional build dependencies and set up environment variables
to make it work.  Refer to the definition of the ``microovn`` part in
``snap/snapcraft.yaml`` for more information.

Test code
~~~~~~~~~

The prerequisites for running linting on the test code are:

* ``make``
* ``shellcheck``

You can install them with:

.. code-block:: none

   sudo apt install make shellcheck

To perform linting, go into the repository's root directory and run:

.. code-block:: none

   make check-lint

Functional tests
----------------

These tests build the MicroOVN snap and use it to deploy the OVN cluster
in LXD containers. This cluster is then used for running functional test
suites.

Satisfy the test requirements
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

There is no need to run tests in dedicated VMs or in isolated environments as
all functional tests run inside containers and no changes are made to the host
running them.

MicroOVN needs to be built prior to running the functional tests. See the
:doc:`Build MicroOVN <building>` page.

Secondly, ensure that you have installed
`Bash Automated Testing System (BATS)`_, a software dependency. Due to the
reliance on its latest features, MicroOVN uses ``BATS`` directly from its
source. If you cloned the MicroOVN repository with submodules (using
``--recurse-submodules`` flag), you are all set and you will have the following
**non-empty** directories:

* ``.bats/bats-assert/``
* ``.bats/bats-core/``
* ``.bats/bats-support/``

If they are empty, you can fetch the submodules with:

.. code-block:: none

   git submodule update --init --recursive

Run functional tests
~~~~~~~~~~~~~~~~~~~~

Once you have your environment set up, running tests is just a matter of
invoking the appropriate ``make`` target. To run all available test suites,
use the ``check-system`` make target:

.. code-block:: none

   make check-system

To run individual test suites you can execute:

.. code-block:: none

   make tests/<name_of_the_test_suite>.bats

By default, functional tests run in LXD containers based on ``ubuntu:lts``
image. This can be changed by exporting environment variable
``MICROOVN_TEST_CONTAINER_IMAGE`` and setting it to a valid LXD image name.

For example:

.. code-block:: none

    export MICROOVN_TEST_CONTAINER_IMAGE="ubuntu:jammy"
    make check-system

Making use of `LXD remotes`_ to spawn containers on a remote cluster or server
is supported through the use of the ``LXC_REMOTE`` `LXD environment`_ variable.

.. code-block:: none

   export LXC_REMOTE=microcloud
   make check-system

.. tip::

   If your hardware can handle it, you can run test suites in parallel by
   supplying ``make`` with ``-j`` argument (e.g. ``make check-system -j4``).
   To avoid interleaving output from these parallel test suites, you can
   specify the ``-O`` argument as well.

Test coverage information
~~~~~~~~~~~~~~~~~~~~~~~~~

When MicroOVN build is configured with the code coverage support via
``microovn/build-aux/environment`` file (see more information about adjusting
MicroOVN build parameters in :doc:`Build MicroOVN <building>` page), system
tests can collect coverage data. All you need to do is export
``MICROOVN_COVERAGE_ENABLED=yes`` environment variable. Example
.. code-block:: none

   # Run all test suites with code coverage
   export MICROOVN_COVERAGE_ENABLED=yes
   make check-system

You can find collected data in the ``.coverage/`` directory, where it's
organised in a ``<test_name>/<container_name>/coverage`` structure. For more
information about the coverage data format and what you can do with it, see
`Go Coverage Documentation`_.

Clean up
~~~~~~~~

Functional test suites will attempt to clean up their containers. However, if
a test crashes, or if it's forcefully killed, you may need to do some manual
cleanup.

If you suspect that tests did not clean up properly, you can list all
containers with:

.. code-block:: none

   lxc list

Any leftover containers will be named according to:
``microovn-<test_suite_name>-<number>``. You can remove them with:

.. code-block:: none

   lxc delete --force <container_name>


.. LINKS
.. _Bash Automated Testing System (BATS): https://bats-core.readthedocs.io/en/stable/
.. _LXD remotes: https://documentation.ubuntu.com/lxd/en/latest/remotes/
.. _LXD environment: https://documentation.ubuntu.com/lxd/en/latest/environment/
.. _golangci-lint: https://golangci-lint.run/
.. _Go Coverage Documentation: https://go.dev/doc/build-cover#working
