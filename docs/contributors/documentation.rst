===============================
Contribute to our documentation
===============================

Contributing to documentation is a great way to get started as a contributor to
open-source projects, no matter your level of experience. Before you start,
please review our :doc:`general guide on contributing to MicroOVN
<index>`.

MicroOVN is growing rapidly, and we would love your help. We welcome, encourage
and appreciate contributions from our user community in the form of
suggestions, fixes and constructive feedback. Whether you are new to MicroOVN
and want to highlight something you found confusing, or you’re an expert and
want to create a how-to guide to help others, we will be happy to work with you
to make our documentation better for everybody.

The MicroOVN documentation is hosted in the `GitHub repository`_, alongside
the rest of the codebase, and is published on `Read the Docs`_.

Diátaxis
--------

Our documentation content, style and navigational structure follows the
`Diátaxis`_ systematic framework for technical documentation authoring. This
framework splits documentation pages into tutorials, how-to guides, reference
material and explanatory text:

* **Tutorials** are lessons that accomplish specific tasks through *doing*.
  They help with familiarity and place users in the safe hands of an instructor.
* **How-to guides** are recipes, showing users how to achieve something,
  helping them get something done. A *How-to* has no obligation to teach.
* **Reference** material is descriptive, providing facts about functionality
  that is isolated from what needs to be done.
* **Explanation** is discussion, helping users gain a deeper or better
  understanding of MicroOVN, as well as how and why MicroOVN functions as it does.

To learn more about our Diátaxis strategy, see
`Diátaxis, a new foundation for Canonical documentation`_.

Improving our documentation and applying the principles of Diátaxis are
on-going tasks. There’s a lot to do, and we don’t want to deter anyone from
contributing to our docs. If you don’t know whether something should be a
tutorial, how-to, reference doc or explanatory text, either ask on the forum or
publish what you’re thinking. Changes are easy to make, and every contribution
helps.

Open Documentation Academy
--------------------------

A key aim of `Canonical Open Documentation Academy`_ initiative is to help
lower the barrier into successful open-source software contribution, by making
documentation into the gateway, and it’s a great way to make your first open
source documentation contributions to MicroOVN.

But even if you’re an expert, we want the academy to be a place to share
knowledge, a place to get involved with new developments, and somewhere you can
ask for help on your own projects.

The best way to get started is with our `documentation task list`_ . Take a
look, bookmark it, and see our `Getting started`_ guide for next steps.

Stay in touch either through the task list, or through one of the following
locations:

* Our `documentation discussion forum`_ on the Ubuntu Community Hub.
* In the `documentation Matrix room`_ for interactive chat.
* `Follow us on Fosstodon`_ for the latest updates and events.

If you’d like to ask us questions outside of our public forums, feel free to
email us at ``docsacademy@canonical.com``.

In addition to the above, we have a weekly Community Hour starting at 16:00 UTC
every Friday. Everyone is welcome, and links and comments can be found on the
`forum post`_.

Finally, subscribe to our `Documentation event calendar`_. We’ll expand our
Community Hour schedule and add other events throughout the year.

Agreements
~~~~~~~~~~

Everyone involved with CODA needs to follow the words and spirit of the
`Ubuntu Code of Conduct v2.0`_. You must also sign and agree to the Canonical
CLA.

Identifying suitable task
~~~~~~~~~~~~~~~~~~~~~~~~~

The academy uses issue labels to give the contributor a glimpse into the task
and what it requires, including the type of task, skills or level of expertise
required, and even the size estimation for the task. You can find tasks of all
sizes in the academy issues list.

From small tasks, such as replacing outdated terminology, checking for broken
links, testing a tutorial or ensuring adherence to the
`Canonical documentation style guide`_\ ; to medium-sized  tasks like,
converting documentation from one format to another, or migrating the contents
of a blog post into the official documentation; to more ambitious tasks, such
as adding a new *How-to* guide, restructuring a group of documents, or
developing new tests and automations.

Completing and closing tasks
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

When a task has been completed to your satisfaction, we’ll ask the contributor
whether they would prefer to merge their work into your project themselves,
or leave this to the project.

Recognition
~~~~~~~~~~~

After successfully completing a task, we’ll give credit to the contributor
and share their success in our forums, on the pages themselves, and in our
news updates and release notes.

Guidance for writing
--------------------

Consistency of writing style in documentation is vital for a good user
experience. To accommodate our audience with a huge variation in experience,
we:

* write with our target audience in mind
* write inclusively and assume very little prior knowledge of the reader
* link or explain phrases, acronyms and concepts that may be unfamiliar, and if
  unsure, err on the side of caution
* adhere to the style guide

Language
~~~~~~~~

MicroOVN documentation currently uses British (GB) English. However, Canonical
recently switched to US English. It is our aim to switch to the US English as
well. Until we completely switch over, the contributions should continue to
use British English.

There are many small differences between UK and US English, but for the most
part, it comes down to spelling. Some common differences are:

* the ``ize`` suffix in preference to ``ise`` (e.g. ``capitalize`` and
  ``capitalise``)
* *our* instead of *or* (as in ``color`` and ``colour``)
* licence as both a verb and noun
* ``catalog`` and ``catalogue``
* dates take the format 1 January 2013, 1-2 January 2025 and 1 January - 2
  February 2025

We use an automated spelling checker that sometimes throws errors about terms
we would like it to ignore:

* If it complains about a file name or a command, enclose the word in double
  backticks (``) to render it as inline code.
* If the word is a valid acronym or a well-known technical term (that should
  not be rendered as code), add it to the spelling exception list,
  ``docs/.custom_wordlist.txt`` (terms should be added in alphabetical order).

Both methods are valid, depending on whether you want the term to be rendered
as normal font, or as inline code (monospaced).

Acronyms
~~~~~~~~

Acronyms should always be capitalised.

They should always be expanded the first time they appear on a page, and then
can be used as acronyms after that. E.g. LSP should be shown as Logical Switch
Port (LSP), and then can be referred to as LSP for the rest of the page.

Links
~~~~~

The first time you refer to a package or other product, you should make it a
link to either that product’s website, or its documentation, or its manpage.

Links should be from reputable sources (such as official upstream docs). Try
not to include blog posts as references if possible. And, always verify that
the links are correct and accurate.

Try to use inline links sparingly. If you have a lot of useful references you
think the reader might be interested in, feel free to include a “Further
reading” section at the end of the page.

Writing style
~~~~~~~~~~~~~

Try to be concise and to-the-point in your writing.

It’s alright to be a bit lighthearted and playful in your writing, but please
keep it respectful, and don’t use emoji (they don’t render well in
documentation, and may not be deemed professional).

It’s also good practice not to assume that your reader will have the same
knowledge as you. If you’re covering a new topic (or something complicated)
then try to briefly explain, or link to supporting explanations of, the things
the typical reader may not know, but needs to (refer to the Diátaxis framework
to help you decide what type of documentation you are writing and the level and
type of information you need to include, e.g. a tutorial may require additional
context but a how-to guide can skip some foundational knowledge - it is safer
to assume some prior knowledge).

Documentation source language
-----------------------------

MicroOVN uses reStructuredText language (reST) for writing the documentation.
You can read about the basic language syntax in the `reStructuredText Primer`_
, or you can see our ``docs/doc-cheat-sheet.rst`` for some examples.

Preview your changes
--------------------

You can verify that your documentation changes render as you expect them by
building the whole documentation set and serve it as a web page locally. To
do that, you can

.. code-block::

   # Change your working directory to the "docs/" if you are in the project root directory
   cd docs/
   # build and serve the documentation as a web page
   make serve

This will start a local web server that serves the current version of the
documentation. If the build was successful, you will see an output like this:

.. code-block::

   The HTML pages are in _build.
   cd "_build"; python3 -m http.server 8000
   Serving HTTP on 0.0.0.0 port 8000 (http://0.0.0.0:8000/) ...

You can either click on the ``http`` link, or open your browser and manually open
the ``http://localhost:8000`` page. From there you can navigate to the
documentation page you changed, and see your changes.

Local Verification
------------------

We have a set of tests that need to pass before we can consider documentation
contribution, similar to tests we expect to pass for the code. These tests will
be executed automatically when you open your pull request on GitHub, but to
speed up the submission and approval process, it is recommended that you run
them locally before you submit your contribution. The tests are defined in the
``docs/Makefile`` and to run them, you can:

.. code-block:: none

   # Change your working directory to the "docs/" if you are in the project root directory
   cd docs/
   # Run spelling check
   make spelling
   # Run link validation check
   make linkcheck
   # Run inclusive language check
   make woke

If all of the check pass without errors, your contribution is ready for submission.

Thank you
---------

We would like to thank you for spending your time to help make the MicroOVN
better. Every contribution, big or small, is important to us, and hopefully a
step in the right direction.


.. LINKS
.. _Canonical documentation style guide: https://docs.ubuntu.com/styleguide/en
.. _Canonical Open Documentation Academy: https://github.com/canonical/open-documentation-academy
.. _Diátaxis: https://diataxis.fr/
.. _Diátaxis, a new foundation for Canonical documentation: https://ubuntu.com/blog/diataxis-a-new-foundation-for-canonical-documentation
.. _Documentation event calendar: https://calendar.google.com/calendar/u/0?cid=Y19mYTY4YzE5YWEwY2Y4YWE1ZWNkNzMyNjZmNmM0ZDllOTRhNTIwNTNjODc1ZjM2ZmQ3Y2MwNTQ0MzliOTIzZjMzQGdyb3VwLmNhbGVuZGFyLmdvb2dsZS5jb20
.. _documentation task list: https://github.com/canonical/open-documentation-academy/issues
.. _documentation discussion forum: https://canonical.com/documentation/open-documentation-academy
.. _documentation Matrix room: https://matrix.to/#/#documentation:ubuntu.com
.. _Getting started: https://discourse.ubuntu.com/t/getting-started/42769
.. _GitHub repository: https://github.com/canonical/microovn
.. wokeignore:rule=master
.. _reStructuredText Primer: https://www.sphinx-doc.org/en/master/usage/restructuredtext/basics.html
.. _Follow us on Fosstodon: https://fosstodon.org/@CanonicalDocumentation
.. _forum post: https://discourse.ubuntu.com/t/documentation-office-hours/42771
.. _Read the Docs: https://canonical-microovn.readthedocs-hosted.com/en/latest/
.. _Ubuntu Code of Conduct v2.0: https://ubuntu.com/community/ethos/code-of-conduct
