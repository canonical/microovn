AUTOM4TE = autom4te
AUTOTEST = $(AUTOM4TE) --language=autotest
BACKEND = lxd
SHELL = bash

COMMON_MACROS_AT = common-macros.at \
		   system-backend-$(BACKEND)-macros.at \
		   system-common-macros.at
SYSTEM_TESTSUITE_AT = system-testsuite.at
SYSTEM_TESTSUITE = system-testsuite

$(SYSTEM_TESTSUITE): package.m4 $(SYSTEM_TESTSUITE_AT) $(COMMON_MACROS_AT)
	$(AUTOTEST) -o $@.tmp $@.at
	mv $@.tmp $@

check-system: $(SYSTEM_TESTSUITE)
	set $(SHELL) '$(SYSTEM_TESTSUITE)';						\
	"$$@" $(TESTSUITEFLAGS) || (test X'$(RECHECK)' = Xyes && "$$@" --recheck)

clean:
	rm -rf $(SYSTEM_TESTSUITE) $(SYSTEM_TESTSUITE).log $(SYSTEM_TESTSUITE).dir
