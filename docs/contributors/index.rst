======================
Contribute to MicroOVN
======================

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
* Each commit must be signed. (See `GitHub documentation about commit signing`_
  )
* The commit message should focus on WHY the change is necessary, we get the
  what and how by looking at the code.
* Include a Signed-off-by header in the commit message. (See
  `Git sign-off documentation`_)
* MicroOVN uses `Launchpad`_ for tracking bugs. If you encounter any issue,
  or have a feature suggestion. Feel free to `open a bug report`_.
* MicroOVN makes use of the GitHub Pull Request workflow.  There is no
  meaningful way to manage interdependencies between GitHub PRs, so we expect
  dependent changes proposed in a single PR reviewed and merged as separate
  commits.
* A proposal for change is not complete unless it contains updates to
  documentation and tests.


Next Steps
----------

.. toctree::
   :maxdepth: 1

   code
   documentation


.. LINKS
.. _Canonical contributor licence agreement: https://ubuntu.com/legal/contributors
.. _GitHub documentation about commit signing: https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification
.. _Git sign-off documentation: https://git-scm.com/docs/git-commit#Documentation/git-commit.txt---signoff
.. _Launchpad: https://bugs.launchpad.net/microovn
.. _open a bug report: https://bugs.launchpad.net/microovn/+filebug
