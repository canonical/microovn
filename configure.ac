# version parameter is unused
AC_INIT([MicroOVN], m4_esyscmd([git describe --always --dirty --abbrev=10]))

AC_ARG_ENABLE([dep-check],
    AS_HELP_STRING([--disable-dep-check], [Don't verify that build dependencies are satisfied]),
    [],[enable_dep_check=yes]
)
if test "$enable_dep_check" = "yes"; then
    AC_CHECK_PROG(HAS_MAKE, [make], [yes], [no])
    if test "$HAS_MAKE" != "yes"; then
        AC_MSG_ERROR(['make' was not found])
    fi

    AC_CHECK_PROG(HAS_SNAPCRAFT, [snapcraft], [yes], [no])
    if test "$HAS_SNAPCRAFT" != "yes"; then
        AC_MSG_ERROR(['snapcraft' was not found])
    fi

    AC_CHECK_PROG(HAS_LXD, [lxd], [yes], [no])
    if test "$HAS_LXD" != "yes"; then
        AC_MSG_ERROR(['lxd' was not found])
    fi
fi

AC_ARG_ENABLE([coverage],
    AS_HELP_STRING([--enable-coverage],
        [Build microovn client and daemon binaries with coverage instrumentation. DO NOT use for production builds!]
    )
)
if test "$enable_coverage" = "yes"; then
    AC_SUBST([GO_COVERAGE_ARG], [-cover])
    AC_SUBST_FILE([GO_COVERAGE_OUTPUT])
    GO_COVERAGE_OUTPUT=$srcdir/snapcraft/coverage_include
else
    AC_SUBST([GO_COVERAGE_ARG], [""])
    AC_SUBST_FILE([GO_COVERAGE_OUTPUT])
    GO_COVERAGE_OUTPUT=/dev/null
fi

AC_CONFIG_FILES([snap/snapcraft.yaml])
AC_CONFIG_FILES([snapcraft/commands/microovn], [chmod +x snapcraft/commands/microovn])
AC_CONFIG_FILES([snapcraft/commands/daemon.start], [chmod +x snapcraft/commands/daemon.start])

AC_OUTPUT
