========================
Contributing to MicroOVN
========================

As an open source project, we welcome contributions of any kind. These can
range from bug reports and code reviews, to significant code or documentation
features.

If you'd like to contribute, you will first need to sign the Canonical
contributor agreement. This is the easiest way for you to give us permission to
use your contributions. In effect, you’re giving us a license, but you still
own the copyright — so you retain the right to modify your code and use it in
other projects.

Please review and sign the `Canonical contributor licence agreement`_.


Contributor guidelines
----------------------

* Each commit should be a logical unit.
* Each commit should pass tests individually to allow bisecting.
* Each commit must be signed.
* The commit message should focus on WHY the change is necessary, we get the
  what and how by looking at the code.
* Include a Signed-off-by header in the commit message.
* MicroOVN makes use of the GitHub Pull Request workflow.  There is no
  meaningful way to manage interdependencies between GitHub PRs, so we expect
  dependent changes proposed in a single PR reviewed and merged as separate
  commits.
* A proposal for change is not complete unless it contains updates to
  documentation and tests.

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

Running tests
~~~~~~~~~~~~~

Prerequisites
^^^^^^^^^^^^^

* lxd (``sudo snap install lxd``)
* GNU parallel (``sudo apt -y install parallel``)
* shellcheck (``sudo apt -y install shellcheck``)
* snapcraft (``sudo snap install snapcraft``)

Lint
^^^^

Go code is linted as part of the snap build process:

    make

To check lint for the test code:

    make check-lint

Functional tests
^^^^^^^^^^^^^^^^

To run the entire suite in serial:

    make check-system

.. LINKS
.. _Canonical contributor licence agreement: https://ubuntu.com/legal/contributors
