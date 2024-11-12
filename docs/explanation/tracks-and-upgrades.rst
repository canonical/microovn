.. _snap channels:

===================================
MicroOVN snap channels and upgrades
===================================

MicroOVN is distributed as a snap. As such, it utilises ``channels`` to manage
and control package upgrades. MicroOVN has dedicated channels for supported LTS
versions of OVN (e.g. ``22.03/stable``, ``24.03/stable``). These dedicated
channels should be used to install production deployments. They are guaranteed
to always contain the same major version of OVN and therefore any automatic
upgrades within the channel won't cause incompatibilities across cluster
members.

Avoid using ``latest`` channel for purposes other then development, testing or
experimentation as it receives updates from the ``main`` development branch.
It can contain experimental features and does not provide guarantees regarding
compatibility of cluster members running different revisions from this channel.


Minor version upgrades
----------------------

Dedicated major version channels of MicroOVN (e.g. ``24.03/stable``) will
automatically receive minor version upgrades whenever the minor upgrade for
the ``OVN`` package becomes available in the ``Ubuntu`` repository. They may
also receive updates regarding MicroOVN itself in form of features or bugfixes
if it's deemed that the backport is warranted.

We try to keep the updates of dedicated stable channels to minimum. Any
automatic upgrades within branch are expected to cause only minimal plane
outage while services restart.


Major version upgrades
----------------------

Starting with version ``22.03``, OVN introduced concept of LTS releases
and started to guarantee the ability to upgrade OVN deployment from one
LTS release to next (`rolling upgrades`_). Therefore, MicroOVN also provides
ability to upgrade deployments from one LTS to another. It tries to take as
much complexity as possible from the process, but it's still potentially
disruptive operation and needs to be triggered by operator manually.

For more information on how to actually perform these upgrades, see
:doc:`How-To: Major Upgrades </how-to/major-upgrades>`

How MicroOVN manages major upgrades
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Upgrades without unnecessary downtime constitutes a challenge for
distributed systems like OVN.

OVN consists of two distributed databases (``Southbound`` and
``Northbound``) and multiple processes (e.g. ``ovn-controller`` or
``ovn-northd``) that rely on ability to read and understand data in these
databases. Major upgrades of OVN often introduce database schema changes and
applying these changes before every host in the deployment is able to
understand them can cause unnecessary outage.

Thanks to the backward compatibility guarantees between LTS versions, new
versions of ``ovn-northd`` and ``ovn-controller`` are able to understand old
database schemas. Therefore we can hold back schema upgrades until every
cluster member is ready for it. And this is what MicroOVN does. It waits until
it receives positive confirmation from every node in the deployment that it's
capable of understanding new database schemas, before triggering database
schema upgrades for ``Southbound`` and ``Northbound`` databases.

.. LINKS
.. _rolling upgrades: https://docs.ovn.org/en/stable/intro/install/ovn-upgrades.html#rolling-upgrade
