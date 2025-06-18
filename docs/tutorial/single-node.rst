===========
Single-node
===========

.. only:: integrated

   .. note::

      MicroCloud users can disregard the instructions on this page, because the MicroCloud setup process handles MicroOVN installation.

This tutorial shows how to install MicroOVN in the simplest way possible.

.. caution::

   A single-node OVN cluster does not have any redundancy (service failover).

Install the software
--------------------

Install MicroOVN on the designated node with the following command:

.. code-block:: none

   sudo snap install microovn

Initialise the cluster
----------------------

.. code-block:: none

   microovn cluster bootstrap

Manage the cluster
------------------

You can interact with OVN using its native commands due to automatically created
snap aliases, for example, to show the contents of the OVN Southbound database:

.. code-block:: none

   ovn-sbctl show
