:orphan:

==========================
Documentation starter pack
==========================

See the `Sphinx and Read the Docs`_ guide for instructions on how to get
started with Sphinx documentation. Then go through the following sections to
use this starter pack to set up your docs repository.

Set up your documentation repository
------------------------------------

You can either create a standalone documentation project based on this
repository or include the files from this repository in a dedicated
documentation folder in an existing code repository.

**Note:** We're planning to provide the contents of this repository as an
installable package in the future, but currently, you need to copy and update
the required files manually.

Standalone documentation repository
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To create a standalone documentation repository, clone this starter pack
repository, `update the configuration <#configure-the-documentation>`_, and
then commit all files to your own documentation repository.

You don't need to move any files, and you don't need to do any special
configuration on Read the Docs.

Documentation in a code repository
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To add documentation to an existing code repository:

#. create a directory called ``docs`` at the root of the code repository
#. populate the above directory with the contents of the starter pack
   repository (with the exception of the ``.git`` directory)
#. copy the file(s) located in the ``docs/.github/workflows`` directory into
   the code repository's ``.github/workflows`` directory
#. in the above file(s), change the values of the ``working-directory`` and
   ``workdir`` fields from "." to "docs"

.. note::

   When configuring RTD itself for your project, the setting **Path for
   .readthedocs.yaml** (under **Advanced Settings**) will need to be given the
   value of "docs/.readthedocs.yaml".

Configure for your project
--------------------------

You must modify some of the default configuration to suit your project. To
simplify keeping your documentation in sync with the starter pack, all custom
configuration is located in the ``custom_conf.py`` file. Go through all
settings in the ``Project information`` section.

Do not modify the centrally maintained ``conf.py`` file.

Configure the header
~~~~~~~~~~~~~~~~~~~~

By default, the header contains elements configured in ``custom_conf.py``. This
includes the product tag, product name (taken from the ``project`` setting ), a
possible link to your product page, and a drop-down menu for "More resources"
that contains possible links to Discourse and GitHub.

You can change any of those links or add further links to the "More resources"
drop-down by editing the ``.sphinx/_templates/header.html`` file. For example,
you might want to add links to announcements, tutorials, guides, or videos that
are not part of the documentation.

Activate/deactivate feedback button
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

A feedback button is included by default, which appears at the top of each page
in the documentation. It redirects users to your GitHub issues page, and
populates an issue for them with details of the page they were on when they
clicked the button.

If your project does not use GitHub issues, set the ``github_issues`` variable
in the ``conf.py`` file to an empty value to disable both the feedback button
and the issue link in the footer.
If you want to deactivate only the feedback button, but keep the link in the
footer, remove the ``github_issue_links.js`` script from the ``conf.py`` file.

Configure included extensions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The starter pack includes a set of extensions that are useful for all
documentation sets. They are pre-configured as needed, but you can customise
their configuration in ``custom_conf.py``.

The following extensions are always included:

- |sphinx-design|_
- |sphinx_tabs.tabs|_
- |sphinx_reredirects|_
- |lxd-sphinx-extensions|_ (``youtube-links``, ``related-links``, ``custom-rst-roles``, and ``terminal-output``)
- |sphinx_copybutton|_
- |sphinxext.opengraph|_
- |myst_parser|_
- |sphinxcontrib.jquery|_
- |notfound.extension|_

You can add further extensions in the ``custom_extensions`` variable in
``custom_conf.py``.

Add custom configuration
~~~~~~~~~~~~~~~~~~~~~~~~

To add custom configurations for your project, see the ``Additions to default
configuration`` and ``Additional configuration`` sections in the
``custom_conf.py``. These can be used to extend or override the common
configuration, or to define additional configuration that is not covered by the
common ``conf.py``.

The following links can help you with additional configuration:

- `Sphinx configuration`_
- `Sphinx extensions`_
- `Furo documentation`_ (Furo is the Sphinx theme we use as our base.)

Change log
----------

See the `change log
<https://github.com/canonical/sphinx-docs-starter-pack/wiki/Change-log>`_ for a
list of relevant changes to the starter pack.

.. LINKS
.. wokeignore:rule=master
.. _`Sphinx configuration`: https://www.sphinx-doc.org/en/master/usage/configuration.html
.. wokeignore:rule=master
.. _`Sphinx extensions`: https://www.sphinx-doc.org/en/master/usage/extensions/index.html
.. _`Furo documentation`: https://pradyunsg.me/furo/quickstart/

.. |sphinx-design| replace:: ``sphinx-design``
.. _sphinx-design: https://sphinx-design.readthedocs.io/en/latest/
.. |sphinx_tabs.tabs| replace:: ``sphinx_tabs.tabs``
.. _sphinx_tabs.tabs: https://sphinx-tabs.readthedocs.io/en/latest/
.. |sphinx_reredirects| replace:: ``sphinx_reredirects``
.. _sphinx_reredirects: https://documatt.gitlab.io/sphinx-reredirects/
.. |lxd-sphinx-extensions| replace:: ``lxd-sphinx-extensions``
.. _lxd-sphinx-extensions: https://github.com/canonical/lxd-sphinx-extensions#lxd-sphinx-extensions
.. |sphinx_copybutton| replace:: ``sphinx_copybutton``
.. _sphinx_copybutton: https://sphinx-copybutton.readthedocs.io/en/latest/
.. |sphinxext.opengraph| replace:: ``sphinxext.opengraph``
.. _sphinxext.opengraph: https://sphinxext-opengraph.readthedocs.io/en/latest/
.. |myst_parser| replace:: ``myst_parser``
.. _myst_parser: https://myst-parser.readthedocs.io/en/latest/
.. |sphinxcontrib.jquery| replace:: ``sphinxcontrib.jquery``
.. _sphinxcontrib.jquery: https://github.com/sphinx-contrib/jquery/
.. |notfound.extension| replace:: ``notfound.extension``
.. _notfound.extension: https://sphinx-notfound-page.readthedocs.io/en/latest/

Next steps
----------

Now that your repository is enabled for doc builds you should:

* rename this present file (``readme.rst``) to ``setup.rst``
* rename file ``working-with-the-docs.rst`` to ``readme.rst``

The new ``readme.rst`` file shows contributors how to work with the
documentation. For a standalone documentation scenario, it will be the
repository's main README file. For the integrated scenario (i.e. documentation
in a code repository), it will remain in the ``docs`` directory where it can be
linked to from your project's main README file.

.. LINKS
.. _Sphinx and Read the Docs: https://canonical-documentation-with-sphinx-and-readthedocscom.readthedocs-hosted.com
