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
.. _strictly confined snap: https://snapcraft.io/docs/snap-confinement
.. _Open Virtual Network: https://www.ovn.org/en/
.. _Code of conduct: https://ubuntu.com/community/ethos/code-of-conduct
.. _GitHub: https://github.com/canonical/microovn
.. _MicroOVN Discourse: https://discourse.ubuntu.com/c/microovn/160
.. _Support: https://discourse.ubuntu.com/c/microovn/support/164
