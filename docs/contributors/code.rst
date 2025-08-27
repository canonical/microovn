======================
Contribute to our code
======================

This page covers topics on how to make/build/test your changes to
the MicroOVN source code.

Get the source code
-------------------

MicroOVN development happens on GitHub. You can find, and contribute to,
its source code in in `our GitHub repository`_.

Build and install MicroOVN from source
--------------------------------------

Build requirements
~~~~~~~~~~~~~~~~~~

MicroOVN is distributed as a snap and the only requirements for building it
are ``Make`` and ``snapcraft``. You can install them with:

.. code-block:: none

   sudo apt install make
   sudo snap install snapcraft --classic

Snapcraft requires ``LXD`` to build snaps. So if your system does not have LXD
installed and initiated, you can check out either `LXD getting started
guides`_ or go with following default setup:

.. code-block:: none

   sudo snap install lxd
   lxd init --auto

Build MicroOVN
~~~~~~~~~~~~~~

To build MicroOVN, go into the repository's root directory and run:

.. code-block:: none

   make

This will produce the ``microovn.snap`` file that can be then used to install
MicroOVN on your system.

.. _build_params:

Adjust build parameters
~~~~~~~~~~~~~~~~~~~~~~~

``snapcraft.yaml`` is by nature a very static build recipe that does not allow
build-time modification without changing the file itself. To achieve some
level of control over MicroOVN builds, we are using a
``microovn/build-aux/environment`` file that is loaded and during the build
process. Environment variables defined in this file can influence properties
of the final build. Currently supported variables are:

* ``MICROOVN_COVERAGE`` (default: ``no``) - When set to ``yes``, MicroOVN binaries
  will be built with coverage instrumentation and output coverage data into
  ``$SNAP_COMMON/data/coverage``.

Install MicroOVN
~~~~~~~~~~~~~~~~

Using the ``microovn.snap`` file created in the previous section, you can
install MicroOVN in this way:

.. code-block:: none

   sudo snap install --dangerous ./microovn.snap

.. note::

   If you are building latest MicroOVN from the ``main`` branch, it's possible
   that it's using a non-stable core snap as its base. In that case, you may
   get a message like this:

   .. code-block:: none

      Ensure prerequisites for "microovn" are available (cannot install snap base "core24": no snap revision available as specified)

   In such a case, you will need to install the required core snap manually
   from the ``edge`` risk level. For example:

   .. code-block:: none

      snap install core24 --edge

   Then repeat the installation step.

You will also need to manually connect required plugs, as ``snapd`` won't
do it automatically for locally installed snaps.

.. code-block:: none

   for plug in firewall-control \
                hardware-observe \
                hugepages-control \
                network-control \
                network-setup-control \
                openvswitch-support \
                process-control \
                system-trace; do \
       sudo snap connect microovn:$plug;done

To verify that all the required plugs are correctly connected to their slots,
you can run:

.. code-block:: none

   snap connections microovn

An example of correctly connected connected plugs would look like this:

.. code-block:: none

   Interface            Plug                          Slot                       Notes
   content              -                             microovn:ovn-certificates  -
   content              -                             microovn:ovn-chassis       -
   content              -                             microovn:ovn-env           -
   firewall-control     microovn:firewall-control     :firewall-control          manual
   hardware-observe     microovn:hardware-observe     :hardware-observe          manual
   hugepages-control    microovn:hugepages-control    :hugepages-control         manual
   microovn             -                             microovn:microovn          -
   network              microovn:network              :network                   -
   network-bind         microovn:network-bind         :network-bind              -
   network-control      microovn:network-control      :network-control           manual
   openvswitch-support  microovn:openvswitch-support  :openvswitch-support       manual
   process-control      microovn:process-control      :process-control           manual
   system-trace         microovn:system-trace         :system-trace              manual

And if the plugs are not connected, the output would look like this:

.. code-block:: none

   Interface            Plug                          Slot                       Notes
   content              -                             microovn:ovn-certificates  -
   content              -                             microovn:ovn-chassis       -
   content              -                             microovn:ovn-env           -
   firewall-control     microovn:firewall-control     -                          -
   hardware-observe     microovn:hardware-observe     -                          -
   hugepages-control    microovn:hugepages-control    -                          -
   microovn             -                             microovn:microovn          -
   network              microovn:network              :network                   -
   network-bind         microovn:network-bind         :network-bind              -
   network-control      microovn:network-control      -                          -
   openvswitch-support  microovn:openvswitch-support  -                          -
   process-control      microovn:process-control      -                          -
   system-trace         microovn:system-trace         -                          -

Tests
-----

The tests mainly focus on functional validation of MicroOVN and how we build
and configure OVN itself.

We expect Go unit tests for pure functions.

For impure functions, i.e. functions with side effects, if you find yourself
redesigning interfaces or figuring out how to mock something to support unit
tests, then stop and consider the following strategies instead:

#. Extract the logic you want to test into pure functions.  When done right the
   side effect would be increased composability, setting you up for future code
   reuse.
#. Contain the remaining functions with side effects in logical units that
   can be thoroughly tested in the integration test suite.

MicroOVN has two types of tests, linter checks and functional tests and this
page will show how to run them.

Linter checks
~~~~~~~~~~~~~

Go code
^^^^^^^

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
^^^^^^^^^

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
~~~~~~~~~~~~~~~~

These tests build the MicroOVN snap and use it to deploy the OVN cluster
in LXD containers. This cluster is then used for running functional test
suites.

Satisfy the test requirements
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

There is no need to run tests in dedicated VMs or in isolated environments as
all functional tests run inside containers and no changes are made to the host
running them.

Ensure that you have installed
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
^^^^^^^^^^^^^^^^^^^^

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
^^^^^^^^^^^^^^^^^^^^^^^^^

When MicroOVN build is configured with the code coverage support via
``microovn/build-aux/environment`` file (see :ref:`Adjust build
parameters <build_params>` section for more info), system
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
^^^^^^^^

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
.. _golangci-lint: https://golangci-lint.run/
.. _Go Coverage Documentation: https://go.dev/doc/build-cover#working
.. _LXD environment: https://documentation.ubuntu.com/lxd/en/latest/environment/
.. _LXD getting started guides: https://documentation.ubuntu.com/lxd/en/latest/getting_started/
.. _LXD remotes: https://documentation.ubuntu.com/lxd/en/latest/remotes/
.. _our GitHub repository: https://github.com/canonical/microovn
