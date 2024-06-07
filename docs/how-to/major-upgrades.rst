======================================
Upgrade MicroOVN across major versions
======================================

MicroOVN is released in channels that signify which version of ``OVN`` it
bundles (e.g. ``22.03/stable`` channel comes with ``OVN 22.03``). These
channels track a specific major version,  and wont upgrade to next major
version on their own. To upgrade to next major version of MicroOVN, you have
to change MicroOVN's snap channel.

In this how-to, we'll upgrade a cluster with four members, running
``MicroOVN 22.03``, to ``MicroOVN 24.03``.

Prepare cluster for upgrade
---------------------------

We start by ensuring that **each** of our cluster members runs MicroOVN from a
channel that precedes the version to which we are upgrading, and that it has
latest upgrades from this channel.

In this example we are upgrading to ``24.03``, so we'll check that our cluster
members run ``22.03``.

.. code-block:: none

   snap info microovn

Example of relevant output from ``snap info``:

.. code-block:: none

   <snipped preceding output>

   snap-id:      llLUDjcLf2hf4zrlty82XqaYTwN4afUP
   tracking:     22.03/stable
   refresh-date: today at 10:07 UTC

   <snipped remaining output>

Next we ensure that MicroOVN runs the latest version in the channel (again on
**each** cluster member):

.. code-block:: none

   sudo snap refresh microovn

As a final preparation step, we'll ensure that all MicroOVN cluster members
are online by running:

.. code-block:: none

   sudo microovn cluster list -f compact

It's sufficient to run this command on a single member. Resulting output
should show status of all members as ``ONLINE``:

.. code-block:: none

   NAME        ADDRESS          ROLE                              FINGERPRINT                             STATUS
   movn1  10.75.224.44:6443   voter     0e359bed39fb0aaedcb730c707b89701abfb0a65ed5e0f9b5ff883a75c914683  ONLINE
   movn2  10.75.224.233:6443  stand-by  b084c2fadd4ca66ffd8fb7e58a1f90f2bbec1fec5ec6d4091eba7e7fbbb66981  ONLINE
   movn3  10.75.224.128:6443  voter     fc9efe07194030ec212a75d32e525a321eb973a0cf071c2bc8841480457a248a  ONLINE
   movn4  10.75.224.11:6443   voter     fa3380a109f48e5bce60ba942cf24617d5db3b4f371dedc6ef732303ada7ed0b  ONLINE

Ensure sufficient election timer
--------------------------------

Upgrade of OVN cluster can be computationally stressful operation, especially
for nodes that run OVN ``central`` services. To prevent cluster members from
missing heartbeats and causing leadership flapping, we recommend setting
``election timer`` of ``Northbound`` and ``Southbound`` databases to at least
``16 seconds``.

To check current values, run following commands:

.. code-block:: none

   # Get OVN Northbound cluster status
   sudo microovn.ovn-appctl -t /var/snap/microovn/common/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound

   # Get OVN Southbound cluster status
   sudo microovn.ovn-appctl -t /var/snap/microovn/common/run/ovn/ovnsb_db.ctl cluster/status OVN_Southbound

Look for ``Election timer:`` in the output of these commands. Value of this
field is expressed in milliseconds.

.. code-block:: none

   <snipped preceding output>

   Last Election won: 56593 ms ago
   Election timer: 16000
   Log: [2, 8]
   Entries not yet committed: 0
   Entries not yet applied: 0
   Connections:
   Disconnections: 0

   <snipped remaining output>

If the value is lower than ``16000``, we recommend gradually increasing it
with:

.. code-block:: none

   # Command example for Northbound election timer increase
   microovn.ovn-appctl -t /var/snap/microovn/common/run/ovn/ovnnb_db.ctl cluster/change-election-timer OVN_Northbound <new_value>

   # Command example for Southbound election timer increase
   microovn.ovn-appctl -t /var/snap/microovn/common/run/ovn/ovnsb_db.ctl cluster/change-election-timer OVN_Southbound <new_value>

``OVN`` wont let you increase the timer by more than twice its current
value, so you will have to proceed gradually.

Upgrade single cluster member
-----------------------------

Now we can proceed with upgrade of individual members in the cluster. The
process itself is very straightforward, we just need to keep an eye on it,
to ensure that it finishes as expected.

We'll start by upgrading single cluster member by running following command
on it:

.. code-block:: none

   sudo snap refresh --channel=24.03/stable microovn

.. important::

   Above command causes restart of MicroOVN and OVN services running on this
   cluster member. This results in temporary data plane outage, for ports
   connected to OVN Chassis located on this member, while services come
   back up and reconfigure datapaths.

After the snap is successfully upgraded, we can check the cluster status with:

.. code-block:: none

   sudo microovn status

The output of the command above will look something like this:

.. code-block:: none

   <snipped preceding output>

   OVN Database summary:
   OVN Southbound: Upgrade or attention required!
   Currently running schema: 20.21.0
   Cluster report (expected schema versions):
   	   movn1: 20.33.0
       movn4: Missing API. MicroOVN needs upgrade
       movn2: Missing API. MicroOVN needs upgrade
       movn3: Missing API. MicroOVN needs upgrade

   OVN Northbound: Upgrade or attention required!
   Currently running schema: 6.1.0
   Cluster report (expected schema versions):
       movn1: 7.3.0
       movn4: Missing API. MicroOVN needs upgrade
       movn3: Missing API. MicroOVN needs upgrade
       movn2: Missing API. MicroOVN needs upgrade

We can see, from the output above, that host ``movn1``, as the only
upgraded member so far, reports that it expects different ``OVN Southbound``
and ``OVN Northbound`` database schema version, as the cluster is currently
running. This is expected and it will remain the case until all the cluster
members are upgraded, at which point the schema upgrade will be triggered.

.. note::

    As the MicroOVN version ``24.03`` is first to support API required to
    report expected schema versions, you will see placeholder messages
    ``Missing API. MicroOVN needs upgrade`` coming from hosts that run
    older MicroOVN versions. Going forward, the output during the future
    upgrades would look something like this:

    .. code-block:: none

       OVN Northbound: Upgrade or attention required!
       Currently running schema: 6.1.0
       Cluster report (expected schema versions):
           movn1: 7.3.0
           movn4: 6.1.0
           movn3: 6.1.0
           movn2: 6.1.0

.. note::

   If you run ``microovn status`` immediately after the snap refresh, you
   may encounter following, or similar, error messages in the output:

   .. code-block:: none

      OVN Database summary:
      Failed to fetch OVN Southbound schema status: failed to fetch OVN Southbound cluster schema status from 'http://control.socket': Internal Server Error
      Error: failed to fetch either Southbound or Northbound database status

   It is expected, as it takes few seconds for the member to reconnect back to
   the cluster. The error message should go away after few seconds.

Continue with cluster upgrade
-----------------------------

Same commands, from the previous section, can be run on the rest of
the cluster members. You should progress one cluster member at a time
and check the output of ``microovn cluster status`` to see if the upgrade
continues as expected.

Final verification
------------------

After the last cluster member is upgraded, MicroOVN will trigger schema
upgrade of OVN databases. This is an asynchronous process that can take
from few seconds, to few minutes, depending on the size of the database.
You can run:

.. code-block:: none

   sudo microovn status

and if the schema upgrade finished successfully, you'll see following output:

.. code-block:: none

   <snipped preceding output>

   OVN Database summary:
   OVN Southbound: OK (20.33.0)
   OVN Northbound: OK (7.3.0)