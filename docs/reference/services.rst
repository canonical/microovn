.. _MicroOVN services:

=================
MicroOVN services
=================

MicroOVN functionality is separated into distinct services that can be easily
controlled via ``microovn enable`` and ``microovn disable``.

This page presents a list of all MicroOVN services. Their descriptions are
for reference only - the user is not expected to interact directly with these
services.

Handling services with enable/disable
-------------------------------------

The status of all services is displayed by running:

.. code-block:: none

   microovn status

``central service``
-------------------

This is responsible for the database control. The database is clustered and uses
the `RAFT <https://docs.openvswitch.org/en/latest/ref/ovsdb.7/#clustered-database-service-model>`_
algorithm for consensus it can handle (n-1)/2 failures, where n is the number of
nodes.

Central is enabled on a new node whenever there are less than 3 nodes running
the central services

This service controls the following `Snap services`_:

- ``microovn.ovn-ovsdb-server-nb``
- ``microovn.ovn-ovsdb-server-sb``
- ``microovn.ovn-northd``

``chassis service``
-------------------

This service controls the ``ovn-controller`` daemon, which is OVN's agent on each
hypervisor and software gateway. It is a distributed component running on the
side of every Open vSwitch instance.
It is enabled by default.

The snap service this controls is ``microovn.chassis``

`switch service``
-------------------

This service ``Open vSwitch`` and ensures its running properly. Much like chassis it
is enabled by default.

The snap service this controls is ``microovn.switch``


Snap services
-------------

The status of all services is displayed by running:

.. code-block:: none

   snap services microovn

``microovn.chassis``
--------------------

This service maps directly to the ``ovn-controller`` daemon.

``microovn.daemon``
-------------------

The main MicroOVN service/process that manages all the other processes. It also
handles communication with other MicroOVN cluster members and provides an API
for the ``microovn`` client command.

``microovn.ovn-ovsdb-server-nb``
--------------------------------

This service maps directly to the ``OVN Northbound`` database/service.

``microovn.ovn-northd``
-----------------------

This service maps directly to the ``ovn-northd`` daemon.

``microovn.ovn-ovsdb-server-sb``
--------------------------------

This service maps directly to the ``OVN Southbound`` database/service.

``microovn.refresh-expiring-certs``
-----------------------------------

This service is a recurring process that runs once a day between ``02:00`` and
``02:30``. It triggers TLS certification reissue for certificates that are
nearing the expiration. For more information see the
:ref:`certificates lifecycle <certificates_lifecycle>`.

``microovn.switch``
-------------------

This services maps directly to the ``ovs-vswitchd`` daemon.
