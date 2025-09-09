========================
MicroOVN Release Process
========================

Release Strategy
----------------

MicroOVN feature development takes place on the "main" branch.

The main branch has ``snapcraft.yaml`` set up to use the ``base`` for the next
core version, and a ``build-base`` set to 'devel', which sources stage packages
from the most recent Ubuntu development release. The test suite will
automatically handle installing the ``base`` from the 'edge' channel when
required.

Stable MicroOVN releases follow the `Ubuntu release cycle`_, and a new stable
version is made shortly after each new Ubuntu LTS release.

The `stable branches`_ are named "branch-YY.MM", where the numbers come from
the corresponding upstream OVN version string, for example: "branch-24.03".

Release Numbering
-----------------

The main component of the MicroOVN snap is OVN, consequently the main component
of the snap version string come from the upstream version string of the OVN
binary embedded in the snap.

The binaries in the snap are sourced from the deb package in the Ubuntu version
corresponding to the Ubuntu Core build base, typically the most recent Ubuntu
LTS release.

Our `build pipeline`_ is configured in Launchpad, and the `MicroOVN snap
recipes`_ are configured to automatically build and publish the snap for
supported channels. Builds are triggered whenever relevant packages in the
source Ubuntu release change, or when the relevant branch in the `MicroOVN
GitHub repository`_ changes.

To allow quick identification of the snap artefact in use, an abbreviated
commit hash from the ``microovn`` Git repository, is appended to the version
string.

The full package version string for all embedded packages can be retrieved by 
issuing the ``microovn --version`` command on a system with the snap installed.

Stable Branches
---------------

We go out of our way to embed logic in the product itself, its test suite and
CI pipeline to avoid manual effort on each new release.

Steps to cut a stable branch:

#. Create a PR named "Prepare for YY.MM" that contains two (or more) commits.

   * First commit

     * Set ``base`` to a stable version of core and remove any
       ``build-base`` statements.
     * Pin any parts with ``source-type`` git to the most recent stable version
       available.

   * Second commit

     * Set ``base`` back to a edge version of core (when available), and add a
       ``build-base`` statement with 'devel' as value.
     * Unpin any parts with ``source-type`` git.

#. Review and merge as separate commits.
#. Create branch ``branch-YY.mm`` using the first commit from step 1 as base.

Build pipeline
--------------

Steps to set up a build pipeline:

#. Go to `Launchpad MicroOVN code`_ repository and ensure required branches
   have been imported.
#. `Create new MicroOVN snap package recipe`_ make sure to populate fields:

   * Owner: ``ubuntu-ovn-eng``.
   * Git repository and branch.
   * Processors: ``amd64``, ``arm64``, ``ppc64el``,  ``riscv64``, ``s390x``.
   * Automatically build when branch changes.
   * Automatically upload to store.

     * Track that corresponds with branch.
     * Risk: ``edge``.

.. LINKS
.. _Ubuntu release cycle: https://ubuntu.com/about/release-cycle
.. _MicroOVN snap recipes: https://launchpad.net/microovn/+snaps
.. _MicroOVN GitHub repository: https://github.com/canonical/microovn.git
.. _Launchpad MicroOVN code: https://code.launchpad.net/microovn
.. _Create new MicroOVN snap package recipe: https://launchpad.net/microovn/+new-snap
