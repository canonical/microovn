================
Security process
================

What is a vulnerability?
------------------------
All vulnerabilities are bugs, but not every bug is a vulnerability.
Vulnerabilities compromise one or more of:

* Confidentiality (personal or corporate confidential data).
* Integrity (trustworthiness and correctness).
* Availability (uptime and service).

If in doubt, please use the process for `reporting a vulnerability`_, and we
will assess whether your report is in fact a security vulnerability, or if it
should be `reported as a bug`_ using the normal bug process.

Reporting a vulnerability
-------------------------
To report a security issue, please email `security@ubuntu.com`_ with a
description of the issue, the steps you took to create the issue, affected
versions, and, if known, mitigations for the issue.

The `Ubuntu Security disclosure and embargo policy`_ contains more information
about what you can expect when you contact us and what we expect from you.

Product lifetime
----------------
The main components of MicroOVN, Open vSwitch (OVS) and Open Virtual Network
(OVN), comes from the Ubuntu distribution.  Releases of MicroOVN in stable
:ref:`snap channels` that align with Ubuntu Long Term Support (LTS) releases,
receive the same level of support throughout the lifetime of the corresponding
Ubuntu LTS release.  Please refer to the `Ubuntu lifecycle and release
cadence`_ documentation for more information.

Tracking vulnerabilities
------------------------
Vulnerabilities, their status, and the state of the analysis or response will
all be tracked through the `Ubuntu CVE tracker`_.

Responding to vulnerabilities
-----------------------------
Vulnerabilities are classified by `priority`_, and the MicroOVN project
guarantees response to all High and Critical severity vulnerabilities, as well
as any `Known Exploited Vulnerability`_.

Security updates will be made available to consumers of stable :ref:`snap
channels` that align with supported Ubuntu Long Term Support (LTS) releases.

The MicroOVN snap is automatically rebuilt by Launchpad whenever there is an
update to the underlying packages in the Ubuntu distribution.

Updated versions of the snap will be put through the MicroOVN functional test
suites before being promoted to stable :ref:`snap channels`.

Information about new builds are made available through the `Snap store`_.

Responsible disclosure
----------------------
We follow the `Ubuntu Security disclosure and embargo policy`_.  Please refer
to the section on `reporting a vulnerability`_.

.. LINKS
.. _security@ubuntu.com: mailto:security@ubuntu.com
.. _Ubuntu Security disclosure and embargo policy: https://ubuntu.com/security/disclosure-policy
.. _reported as a bug: https://bugs.launchpad.net/microovn/+filebug
.. _Ubuntu lifecycle and release cadence: https://ubuntu.com/about/release-cycle
.. _Ubuntu CVE tracker: https://ubuntu.com/security/cves
.. _priority: https://ubuntu.com/security/cves/about#priority
.. _Known Exploited Vulnerability: https://www.cisa.gov/known-exploited-vulnerabilities-catalog
.. _Snap store: https://snapcraft.io/store
