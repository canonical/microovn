========
MicroOVN
========

``MicroOVN`` is a snap-based distribution of OVN - `Open Virtual Network`_.

It allows users to deploy an OVN cluster with just a few commands. Aside from
the regular OVN packages, ``MicroOVN`` comes bundled with a CLI utility
(``microovn``) that facilitates operational management. In particular, it
simplifies the task of adding/removing cluster members and incorporates status
checking out of the box.

Besides the ease of deployment and a convenient CLI tool, another benefit of
``MicroOVN`` is in its self-contained nature: it is distributed as a `strictly
confined snap`_. This means that it can be easily upgraded/downgraded/removed
without affecting the host system.

``MicroOVN`` caters to a wide range of user and environment types. It lowers
the barrier of entry to OVN for people that are less familiar with it by
automating much of the deployment process. It also provides a fully fledged,
unrestricted OVN deployment that is suitable for both development and
production environments.

.. toctree::
   :hidden:
   :maxdepth: 2

   how-to/index
   tutorial/index
   reference/index

.. LINKS
.. _strictly confined snap: https://snapcraft.io/docs/snap-confinement
.. _Open Virtual Network: https://www.ovn.org/en/
