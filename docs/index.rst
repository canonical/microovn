========
MicroOVN
========

``MicroOVN`` is a snap-based distribution of ``OVN (Open Virtual Network)``.

It allows users to easily deploy an ``OVN`` cluster with just a few commands.
Aside from regular ``OVN`` packages, ``MicroOVN`` comes bundled with CLI
utility (``microovn``) that facilitates deployment management. Among other
things, it allows adding or removing cluster members and status checking.

Besides the ease of deployment and convenient CLI, another benefit of
``MicroOVN`` is its self-contained nature. It is distributed as a strictly
confined snap which means that it can be easily upgraded/downgraded/removed
without affecting host system.

``MicroOVN`` can be useful for wide range of users. It lowers a barrier of
entry to ``OVN`` for people that are not yet familiar with it by automating as
much of a deployment process as possible. At the same time, the aim is for
``MicroOVN`` to provide fully fledged ``OVN`` deployment not restricted in any
way and suitable for production environment.

.. toctree::
   :hidden:
   :maxdepth: 2

   how-to/index
