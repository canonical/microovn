===============
Service Control
===============

Service control refers to the ability to specify which OVN services run
on any given node in the cluster. The list of services can be found here
:doc:`Services Reference </reference/services>` and they are responsible for
handling core functionality.

The services of the each cluster node can be specified either during the
initialisation of the node, or after the deployment via MicroOVN's CLI.

The approach of using snap CLI to enable/disable services is not recommended,
because it does not update the desired state or handle joining clusters and
configuring the service properly.

Change services on a deployed cluster
-------------------------------------

MicroOVN offers ``disable`` and ``enabled`` subcommands that can change which
services are active on the current cluster node.

.. note::

   This assumes you have MicroOVN installed and a clustered across three of more
   nodes. These nodes will be referred to as ``first``, ``second`` and ``third``
   respectively.

Disable a MicroOVN service
~~~~~~~~~~~~~~~~~~~~~~~~~~

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


Enable a MicroOVN service
~~~~~~~~~~~~~~~~~~~~~~~~~

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
~~~~

Typically the most common use case of this will be to control the nodes the
central services are running on and to increase the number of central services
beyond the default of 3.

Specify services during the cluster deployment
----------------------------------------------

The default selection of services for the node can be adjusted via the interactive
``microovn init`` command during the deployment (instead of using ``bootstrap`` and
``join`` methods). The user is asked question:

.. code-block:: none

   Please select comma-separated list services you would like to enable on this node (central/chassis/switch) or let MicroOVN automatically decide (auto) [default=auto]:

Here, they can either define the desired services as a comma-separated string or select
``auto`` option which falls back to the default behaviour. Leaving this option empty
has same effect as selecting ``auto``.

.. note::

   The default behaviour for selecting services is to always enable ``switch``
   and ``chassis`` services. The ``central`` service is enabled only if configuration
   option :doc:`ovn.central-ips </reference/config/ovn-central-ips>` is not set and
   there are less than 3 nodes with ``central`` service enabled in the cluster.
