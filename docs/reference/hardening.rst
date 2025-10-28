===============================
Configuration safety guidelines
===============================

MicroOVN is a very opinionated way to deploy OVN. It enforces TLS encryption and
authentication on its network endpoints, and it tries to use sane defaults
wherever possible. In general, it does not provide many ways to deviate from
the standard configuration, but this section will highlight those places where
it does and where the security can be improved by the user manually.

BGP integration
---------------

MicroOVN provides a way to integrate OVN natively with BGP routers on the
external networks. See :doc:`Configure OVN BGP integration </how-to/bgp>`
page for more information. When the integration is enabled with the ``--asn``
option specified, MicroOVN will auto-configure a `BIRD 3`_ BGP service to listen
on connections from the physical external network. This auto-configured BGP
daemon has a very lax security settings, most importantly it:

* doesn't perform peer authentication (see `RFC 2385`_)
* doesn't employ RPKI to validate route advertisements (see `RFC 6480`_)
* doesn't apply any route filtering on learned routes
* does connect to the first peer it finds on the external link

BGP security is a very broad topic that's out of scope for this document, but
the above points should cover basics when deploying BGP daemons in an
environment where the peers can't be necessarily trusted.

If the user desires any of the above security features, they are advised to
omit the ``--asn`` option when enabling the BGP integration. This will allow
them to bind any external BGP daemon to the interface inside the VRF created
by the MicroOVN. Then they will be able to tailor the daemon configuration
to their specific security needs.

.. LINKS
.. _BIRD 3: https://bird.network.cz/?get_doc&f=bird.html&v=30
.. _RFC 2385: https://datatracker.ietf.org/doc/html/rfc2385
.. _RFC 6480: https://datatracker.ietf.org/doc/html/rfc6480
