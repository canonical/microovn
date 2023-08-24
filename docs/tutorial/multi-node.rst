===========
Multi-node
===========

This tutorial shows how to install a 3-node MicroOVN cluster. It will deploy an
OpenStack 2023.1 (Antelope) cloud.

One big advantage of a multi-node cluster is that it provides redundancy
(service failover). A 3-node deployment can tolerate up to one node failure.

Requirements
------------

You will need three (virtual or physical) machines that can communicate with
each other over the network. They will be known here as ``node-1``, ``node-2``,
and ``node-3``.

Install the software
--------------------

Install MicroOVN on **each** of the designated nodes with the following
command:

.. code-block:: none

   sudo snap install microovn

Initialise the cluster
----------------------

On **node-1**, initialise the cluster:

.. code-block:: none

   microovn cluster bootstrap

Generate access tokens
----------------------

On **node-1**, generate access tokens for the other two nodes (cluster
members). These will be needed to join these nodes to the cluster.

Let this token be for node-2:

.. code-block:: none

   microovn cluster add node-2

The output will be a special string such as:
``eyJuYW1lIjoibm9kZS0yIiwic2VjcmV0IjoiMzBlM...``.

Let this token be for node-3:

.. code-block:: none

   microovn cluster add node-3

Similarly, a string will be sent to the screen:
``eyJuYW1lIjoibm9kZS0zIiwic2VjcmV0IjoiZmZhY...``.

Complete the cluster
--------------------

Join node-2 and node-3 to the cluster using their assigned access tokens.

On **node-2**:

.. code-block:: none

   microovn cluster join eyJuYW1lIjoibm9kZS0yIiwic2VjcmV0IjoiMzBlM...

On **node-3**:

.. code-block:: none

   microovn cluster join eyJuYW1lIjoibm9kZS0zIiwic2VjcmV0IjoiZmZhY...

Now all three nodes are joined to the cluster.

Manage the cluster
------------------

You can interact with OVN using its native commands prefaced with the string
``microovn.``. For example, to show the contents of the OVN Southbound
database:

.. code-block:: none

   microovn.ovn-sbctl show

The cluster can be managed from any of its nodes.
