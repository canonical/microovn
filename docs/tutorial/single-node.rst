===========
Single-node
===========

This tutorial shows how to install MicroOVN in the simplest way possible. It
will deploy an OpenStack 2023.1 (Antelope) cloud.

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

You can interact with OVN using its native commands prefaced with the string
``microovn.``. For example, to show the contents of the OVN Southbound
database:

.. code-block:: none

   microovn.ovn-sbctl show
