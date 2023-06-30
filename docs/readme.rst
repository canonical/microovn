Documentation starter pack
==========================

See the `Sphinx and Read the Docs <https://canonical-documentation-with-sphinx-and-readthedocscom.readthedocs-hosted.com/>`_ guide for instructions on how to get started with Sphinx documentation.

Then go through the following sections to use this starter pack to set up your docs repository.

Set up your documentation repository
------------------------------------

You can either create a standalone documentation project based on this repository or include the files from this repository in a dedicated documentation folder in an existing code repository.

**Note:** We're planning to provide the contents of this repository as an installable package in the future, but currently, you need to copy and update the required files manually.

Standalone documentation repository
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To create a standalone documentation repository, clone this starter pack repository, `update the configuration <#configure-the-documentation>`_, and then commit all files to your own documentation repository.

You don't need to move any files, and you don't need to do any special configuration on Read the Docs.

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

Getting started
---------------

There are make targets defined in the ``Makefile`` that do various things. To
get started, we will:

* install prerequisite software
* view the documentation

Install prerequisite software
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To install the prerequisites:

.. code-block:: none

   make install

This will create a virtual environment (``.sphinx/venv``) and install
dependency software (``.sphinx/requirements.txt``) within it.

A complete set of pinned, known-working dependencies is included in
``.sphinx/pinned-requirements.txt``.

View the documentation
~~~~~~~~~~~~~~~~~~~~~~

To view the documentation:

.. code-block:: none

   make run

This will do several things:

* activate the virtual environment
* build the documentation
* serve the documentation on **127.0.0.1:8000**
* rebuild the documentation each time a file is saved
* send a reload page signal to the browser when the documentation is rebuilt

The ``run`` target is therefore very convenient when preparing to submit a
change to the documentation. For a more manual approach, to strictly build and
serve content, explore the ``html`` and ``serve`` make targets, respectively.

Configure the documentation
---------------------------

You must modify some of the default configuration to suit your project.

Add your project information
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In ``conf.py``, check or edit the following settings in the *Project information* section:

* ``project``
* ``author``
* ``copyright`` - in most cases, replace the first ``%s`` with the year the project started
* ``release`` - only required if you're actually using release numbers
  (beyond the scope of this guide, but you can also use Python to pull this
  out of your code itself)
* ``ogp_site_url`` - the URL of the documentation output (needed to generate a preview when linking from another website)
* ``ogp_site_name`` - the title you want to use for the documentation in previews on other websites (by default, this is set to the project name)
* ``ogp_image`` - an image that will be used in the preview on other websites
* ``html_favicon`` - the favicon for the documentation (circle of friends by default)

In the ``html_context`` variable, update the following settings:

* ``discourse_prefix`` - the URL to the Discourse instance your project uses (needed to add links to posts using the ``:discourse:`` metadata at the top of a file)
* ``github_url`` - the link to your GitHub repository (needed to create the Edit link in the footer and the feedback button)
* ``github_version`` - the branch that contains this version of the documentation
* ``github_folder`` - the folder that contains the documentation files

Save ``conf.py``.

Configure the spelling check
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If your documentation uses US English instead of UK English, change this in the
``.sphinx/spellingcheck.yaml`` file.

To add exceptions for words the spelling check marks as wrong even though they are correct, edit the ``.wordlist.txt`` file.

Configure the link check
~~~~~~~~~~~~~~~~~~~~~~~~

If you have links in the documentation that you don't want to be checked (for
example, because they are local links or give random errors even though they
work), you can add them to the ``linkcheck_ignore`` variable in the ``conf.py``
file.

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
