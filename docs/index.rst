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

---------

In this documentation
---------------------

..  grid:: 1 1 2 2

   ..  grid-item:: :doc:`Tutorial <tutorial/index>`

       **Start here**: a hands-on introduction to MicroOVN for new users

   ..  grid-item:: :doc:`How-to guides <how-to/index>`

      **Step-by-step guides** covering key operations and common tasks

.. grid:: 1 1 2 2

   .. grid-item:: :doc:`Reference <reference/index>`

      **Technical information** - specifications, APIs, architecture

---------

Project and community
---------------------

MicroOVN is a member of the Ubuntu family. It’s an open source project that
warmly welcomes community projects, contributions, suggestions, fixes and
constructive feedback.

* We follow the Ubuntu community `Code of conduct`_
* Contribute to the project on `GitHub`_ (documentation contributions go under
  the :file:`docs` directory)
* GitHub is also used as our bug tracker
* To speak with us, you can find us in our `MicroOVN Discourse`_ category. Use
  the `Support`_ sub-category for technical assistance.

.. toctree::
   :hidden:
   :maxdepth: 2

   how-to/index
   tutorial/index
   reference/index

.. LINKS
.. _Code of conduct: https://ubuntu.com/community/ethos/code-of-conduct
.. _GitHub: https://github.com/canonical/microovn
.. _MicroOVN Discourse: https://discourse.ubuntu.com/c/microovn/160
.. _Support: https://discourse.ubuntu.com/c/microovn/support/164
