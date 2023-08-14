# Contributing to MicroOVN

As an open source project, we welcome contributions of any kind. These can
range from bug reports and code reviews, to significant code or documentation
features.

If you'd like to contribute, you will first need to sign the Canonical
contributor agreement. This is the easiest way for you to give us permission to
use your contributions. In effect, you’re giving us a license, but you still
own the copyright — so you retain the right to modify your code and use it in
other projects.

The agreement can be found, and signed, here:
https://ubuntu.com/legal/contributors

## Contributor guidelines

- Each commit should be a logical unit.
- Each commit should pass tests individually to allow bisecting.
- The commit message should focus on WHY the change is necessary, we get the
  what and how by looking at the code.
- Include a Signed-off-by header in the commit message.
- MicroOVN makes use of the GitHub Pull Request workflow.  There is no
  meaningful way to manage interdependencies between GitHub PRs, so we expect
  dependent changes proposed in a single PR reviewed and merged as separate
  commits.
- A proposal for change is not complete unless it contains updates to
  documentation and tests.

## Tests

The tests mainly focus on functional validation of MicroOVN and how we build
and configure OVN itself.

Golang unit tests are also welcome, but if you find yourself redesigning
interfaces or figuring out how to mock something to support unit testing, then
stop and express your test by executing the program through the testsuite
instead.

### Running tests

#### Prerequisites

* lxd (`sudo snap install lxd`)
* GNU parallel (`sudo apt -y install parallel`)
* shellcheck (`sudo apt -y install shellcheck`)

#### Lint

    make check-lint

#### Functional tests

To run the entire suite in serial:

    make check-system
