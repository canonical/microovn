===============
Service Control
===============

Service control refers to the set of commands for enabling and disabling a
given MicroOVN service. MicroOVN has a set of services referred to here
:doc:`Services Reference </reference/services>`, which are responsible for
handling core functionality.

You can disable services manually using snap, but the service control does not
update the desired state or handle joining clusters and configuring the service
properly, hence the strong reasoning to interact with services through this
method.

.. note::

   This assumes you have MicroOVN installed and a clustered across three of more
   nodes. These nodes will be referred to as ``first``, ``second`` and ``third``
   respectively.

Disabling a MicroOVN service
----------------------------

Disabling a MicroOVN service will configure it to not start automatically at
boot and stop the service if it is running.

run on ``first``:

.. code-block:: none

   microovn disable switch

.. code-block:: none

   Service switch disabled

To validate that this has worked, we can query the status of MicroOVN and check
which services are enabled. We should find that all nodes have central, chassis
and switch, except first having only central and chassis. This shows the
disabling worked

run on ``first``:

.. code-block:: none

   microovn status

.. code-block:: none

   MicroOVN deployment summary:
   - first (10.190.155.5)
   Services: central, chassis
   - second (10.190.155.174)
   Services: central, chassis, switch
   - third (10.190.155.55)
   Services: central, chassis, switch
   OVN Database summary:
   OVN Northbound: OK (7.3.0)
   OVN Southbound: OK (20.33.0)

The other nodes have also been informed of this change to the service placement
and when queried will confirm that switch is disabled on first from their
perspective too.

run on ``second``:

.. code-block:: none

   microovn status

.. code-block:: none

   MicroOVN deployment summary:
   - first (10.190.155.5)
   Services: central, chassis
   - second (10.190.155.174)
   Services: central, chassis, switch
   - third (10.190.155.55)
   Services: central, chassis, switch
   OVN Database summary:
   OVN Northbound: OK (7.3.0)
   OVN Southbound: OK (20.33.0)


Enabling a MicroOVN service
---------------------------

Enabling a MicroOVN service will configure it to start automatically at boot and
if the service is not running, start it.

run on ``first``:

.. code-block:: none

   microovn enable switch

.. code-block:: none

   Service switch enabled

.. note::

   If the switch service is enabled you may get an error, this is fine.

This will enable the switch service in MicroOVN, This can be shown through the
listing of system services owned by MicroOVN. As mentioned in the disable
section, these do not always translate directly to a MicroOVN service, but in
this case it does.

run on ``first``:

.. code-block:: none

   microovn status

.. code-block:: none

   MicroOVN deployment summary:
   - first (10.190.155.5)
   Services: central, chassis, switch
   - second (10.190.155.174)
   Services: central, chassis, switch
   - third (10.190.155.55)
   Services: central, chassis, switch
   OVN Database summary:
   OVN Northbound: OK (7.3.0)
   OVN Southbound: OK (20.33.0)

You should be able to see here that the service is running and enabled on
startup. The other nodes are also aware of this as if you query the status you
will see it there and running.

run on ``second``:

.. code-block:: none

   microovn status

.. code-block:: none

   MicroOVN deployment summary:
   - first (10.190.155.5)
   Services: central, chassis, switch
   - second (10.190.155.174)
   Services: central, chassis, switch
   - third (10.190.155.55)
   Services: central, chassis, switch
   OVN Database summary:
   OVN Northbound: OK (7.3.0)
   OVN Southbound: OK (20.33.0)

Uses
----

Typically the most common use case of this will be to control the nodes the
central services are running on and to increase the number of central services
beyond the default of 3.
