=============================
Inspect a MicroOVN deployment
=============================

The ``microovn inspect`` command checks the health of a MicroOVN deployment.
It compares the configured services with their runtime state, checks the OVN
databases, and looks for differences in the generated OVN environment across
cluster members.

The command is read-only. It does not change services, files, OVN database
records, or cluster membership. Remediation shown in the output is guidance
for the operator and is never run automatically.

Run an inspection
-----------------

Run the command with elevated privileges on any MicroOVN cluster member:

.. code-block:: none

   sudo microovn inspect

The default output identifies the node and inspection scope, describes the
Northbound and Southbound databases, and lists findings which need attention.
Passing checks are omitted. This is equivalent to ``--format=text`` (or
``-f text``).

Use ``--verbose`` (or ``-v``) to include passing checks, node-level details,
collection errors, and remediation guidance:

.. code-block:: none

   sudo microovn inspect --verbose

Use JSON output when the report will be consumed by another program:

.. code-block:: none

   sudo microovn inspect --format=json

JSON reports always contain the complete result set, so ``--verbose`` cannot
be combined with ``--format=json``. The ``schema_version`` field identifies
the report format. Consumers should allow additional optional fields within a
schema version.

An inspection has an overall 30-second time limit. Collection from an
individual remote member is limited to five seconds. Database probing has its
own bounded convergence window within the overall limit.

Understand the result
---------------------

Each check has one of four statuses:

``PASS``
   The available evidence satisfies the check.

``WARNING``
   MicroOVN is operating, but a known condition is degraded or carries risk.

``FAIL``
   The available evidence proves that a condition is unhealthy.

``UNKNOWN``
   MicroOVN could not collect enough evidence to determine health.

The summary counts every status. Its overall status uses this precedence:
``FAIL``, ``UNKNOWN``, ``WARNING``, then ``PASS``. This keeps an incomplete
inspection visible even when other checks have passed.

The command uses these exit codes:

``0``
   Every check passed.

``1``
   At least one check returned ``WARNING``, ``FAIL``, or ``UNKNOWN``.

   ``2``
   The arguments were invalid, or inspection failed before it could produce a
   report. This includes running the command on a member which has not joined
   or set up a MicroOVN cluster.

An unavailable cluster member is reported as ``UNKNOWN``. Results collected
from other members are still included in the report.

Choose a cluster member
-----------------------

Run the command on a voting member for an authoritative cluster-wide report.
A standby or spare member can provide local evidence, but it cannot establish
the state of the whole cluster. Its report contains an authority warning and
``UNKNOWN`` results for checks which require cluster-wide state, so it exits
with code ``1``.

The central topology check also reports a warning when the deployment has one
or two central members, or an even non-zero number of central members. A
deployment with no managed central service is not warned because its central
databases may be managed externally.

Checks performed
----------------

The inspection covers these areas:

* Central topology - checks the number of members configured with the
  ``central`` service.
* Service runtime - compares configured services with the snap daemons running
  on each member. A configured service which is inactive or disabled is a
  failure. An active or enabled daemon without a corresponding configured
  service is a warning.
* OVN databases - checks Northbound and Southbound schema agreement,
  connectivity, and ``nb_cfg`` convergence.
* DHCP metadata routes - checks that logical switch ports with DHCP options
   have a classless static route to the instance metadata service.
* Environment convergence - compares the generated ``ovn.env`` entries across
  reachable members.

Environment values are hashed on the member where they are collected. Raw
values are not returned or printed. The hashes prevent accidental disclosure,
but they are not a substitute for secret storage: a guessed low-entropy value
can be confirmed offline by hashing it and comparing the result.
