=======================
Downscaling the cluster
=======================

Impact
------

Downscaling can have an adverse effect on the availability and resiliency of
the cluster, especially when a member is being removed that runs an OVN central
service (OVN SB, OVN NB, OVN Northd).

OVN uses the `Raft consensus algorithm`_ for cluster management, which has a
fault tolerance of up to ``(N-1)/2`` members. This means that fault resiliency
will be lost if a three-node cluster is reduced to two nodes.

Monitoring
----------

You can watch logs on the departing member for indications of removal failures
with:

.. code-block:: none

   snap logs -f microovn.daemon

Any issues that arise during the removal process will need to be resolved
manually.

Remove a cluster member
-----------------------

To remove a cluster member:

.. code-block:: none

   microovn cluster remove <member_name>

The value of ``<member_name>`` is taken from the **Name** column in the output
of the :command:`cluster list` command.

Any chassis components (``ovn-controller`` and ``ovs-vswitchd``) running on the
member will first be stopped and disabled (prevented from starting). For a
member with central components present (``microovn.central``), the Northbound
and Southbound databases will be gracefully removed.

Verification
------------

Upon removal, check the state of OVN services to ensure that the member was
properly removed.

.. code-block:: none

   # Check status of OVN SB cluster
   ovn-appctl -t /var/snap/microovn/common/run/central/ovnsb_db.ctl cluster/status OVN_Southbound

   # Check status of OVN NB cluster
   ovn-appctl -t /var/snap/microovn/common/run/central/ovnnb_db.ctl cluster/status OVN_Northbound

   # Check registered chassis
   ovn-sbctl show

Data preservation
-----------------

MicroOVN will back up selected data directories into the timestamped location
:file:`/var/snap/microovn/common/backup_<timestamp>/`. These backups will
include:

* logs
* OVN database files
* OVS database file
* issued certificates and keys

.. LINKS
.. _Raft consensus algorithm: https://raft.github.io
