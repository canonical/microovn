.. _MicroOVN services:

=================
MicroOVN services
=================

This page presents a list of all MicroOVN services. Their descriptions are
for reference only - the user is not expected to interact directly with these
services.

The status of all services is displayed by running:

.. code-block:: none

   snap services microovn

``microovn.central``
--------------------

.. warning::

   The ``microovn.central`` service is deprecated and will be removed in a
   future release.

This is a transitional service. Starting this service will start and enable
multiple services:

* ``microovn.ovn-ovsdb-server-nb``
* ``microovn.ovn-ovsdb-server-sb``
* ``microovn.ovn-northd``

However this service is not capable of stopping these child services so its
usage is strongly discouraged. Users should use individual services instead.

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
