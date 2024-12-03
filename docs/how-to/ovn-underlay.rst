===================================
Create  custom OVN underlay network
===================================

The underlay network is the physical network infrastructure that provides connectivity between the nodes in an OVN deployment and is responsible for carrying encapsulated traffic between OVN components through Geneve (`Generic Network Virtualization Encapsulation`) tunnels.
This allows the virtual network traffic to be transported over the physical underlay network.
Now, by default, MicroOVN uses the hostname of a cluster member as a Geneve endpoint to set up the underlay network, but it is also possible to use custom Geneve endpoints for the cluster members.


Set up the underlay network
~~~~~~~~~~~~~~~~~~~~~~~~~~~

To tell MicroOVN to use the underlay network, you need to provide the IP address of the underlay network interface on each node.
Let us assume that we want to create a three-node OVN cluster and that each node has a dedicated interface ``eth1`` with an IP address. Let says that ``10.0.1.{2,3,4}`` are the respective addresses on the ``eth1`` interface on each node.
You can set the underlay network IP address in the :command:`init` :

.. code-block:: none

   microovn init

Example of the interaction:

.. code-block:: none

   root@micro1:~# microovn init
   Please choose the address MicroOVN will be listening on [default=10.242.68.93]:
   Would you like to create a new MicroOVN cluster? (yes/no) [default=no]: yes
   Please choose a name for this system [default=micro1]:
   Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]: yes
   Please enter the custom encapsulation IP address for this member: 10.0.1.2
   Would you like to add additional servers to the cluster? (yes/no) [default=no]: yes
   What's the name of the new MicroOVN server? (empty to exit): micro2
   eyJzZWNyZXQiOiJmOWU1OWU0N2Q1M2E0ZjJlYTYzNWYwMzIzYTE5ZTgyMjEyMzA3ZmJmY2U5OTRiOTk3NzQ4ZTAyM2VmOGEyN2MyIiwiZmluZ2VycHJpbnQiOiJlZGY0MzEzY2ZkOWFiMTdmYWIwZTZkMmE3MWZiNGZlM2U5M2RjZTBjNzNhYTQ4NWI3ZTk2Zjk2YzBhZmZlOWU2Iiwiam9pbl9hZGRyZXNzZXMiOlsiMTAuMjQyLjY4LjkzOjY0NDMiXX0=
   What's the name of the new MicroOVN server? (empty to exit): micro3
   eyJzZWNyZXQiOiI5MWYzODUyZTA4ZjQyOWQxNGE2Y2JiZWI0NGNmODkyMjRjNzUzZjU1NjYzYTY3MjE5ZjZkMmVhOGM0MTdhM2YxIiwiZmluZ2VycHJpbnQiOiJlZGY0MzEzY2ZkOWFiMTdmYWIwZTZkMmE3MWZiNGZlM2U5M2RjZTBjNzNhYTQ4NWI3ZTk2Zjk2YzBhZmZlOWU2Iiwiam9pbl9hZGRyZXNzZXMiOlsiMTAuMjQyLjY4LjkzOjY0NDMiXX0=
   What's the name of the new MicroOVN server? (empty to exit):

   root@micro2:~# microovn init
   Please choose the address MicroOVN will be listening on [default=10.242.68.13]:
   Would you like to create a new MicroOVN cluster? (yes/no) [default=no]: no
   Please enter your join token: eyJzZWNyZXQiOiJmOWU1OWU0N2Q1M2E0ZjJlYTYzNWYwMzIzYTE5ZTgyMjEyMzA3ZmJmY2U5OTRiOTk3NzQ4ZTAyM2VmOGEyN2MyIiwiZmluZ2VycHJpbnQiOiJlZGY0MzEzY2ZkOWFiMTdmYWIwZTZkMmE3MWZiNGZlM2U5M2RjZTBjNzNhYTQ4NWI3ZTk2Zjk2YzBhZmZlOWU2Iiwiam9pbl9hZGRyZXNzZXMiOlsiMTAuMjQyLjY4LjkzOjY0NDMiXX0=
   Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]: yes
   Please enter the custom encapsulation IP address for this member: 10.0.1.3

   root@micro3:~# microovn init
   Please choose the address MicroOVN will be listening on [default=10.242.68.170]:
   Would you like to create a new MicroOVN cluster? (yes/no) [default=no]:
   Please enter your join token: eyJzZWNyZXQiOiI5MWYzODUyZTA4ZjQyOWQxNGE2Y2JiZWI0NGNmODkyMjRjNzUzZjU1NjYzYTY3MjE5ZjZkMmVhOGM0MTdhM2YxIiwiZmluZ2VycHJpbnQiOiJlZGY0MzEzY2ZkOWFiMTdmYWIwZTZkMmE3MWZiNGZlM2U5M2RjZTBjNzNhYTQ4NWI3ZTk2Zjk2YzBhZmZlOWU2Iiwiam9pbl9hZGRyZXNzZXMiOlsiMTAuMjQyLjY4LjkzOjY0NDMiXX0=
   Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]: yes
   Please enter the custom encapsulation IP address for this member: 10.0.1.4

Now, the MicroOVN cluster is configured to use the underlay network with the IP addresses ``10.0.1.{2,3,4}`` on each node as tunnel endpoint for the encapsulated traffic.
To verify that the underlay network is correctly configured, you can check the IP of OVN Geneve tunnel endpoint on each node:

.. code-block:: none

   root@micro1:~# ovs-vsctl get Open_vSwitch . external_ids:ovn-encap-ip
   "10.0.1.2"

   root@micro2:~# ovs-vsctl get Open_vSwitch . external_ids:ovn-encap-ip
   "10.0.1.3"

   root@micro3:~# ovs-vsctl get Open_vSwitch . external_ids:ovn-encap-ip
   "10.0.1.4"

