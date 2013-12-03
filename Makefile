all:

install:
	mkdir -p $(DESTDIR)/usr/lib/spack
	mkdir -p $(DESTDIR)/usr/bin
	mkdir -p $(DESTDIR)/etc/spack/repos/
	install -c src/forge/forge.sh $(DESTDIR)/usr/bin/forge
	install -c src/wield/wield.sh $(DESTDIR)/usr/bin/wield
	install -c src/spack/spack.sh $(DESTDIR)/usr/bin/spack
	install -c src/libspack.sh    $(DESTDIR)/usr/lib/spack/libspack
	install -c conf/spack.conf     $(DESTDIR)/etc/spack/spack.conf
	install -c conf/core.repo     $(DESTDIR)/etc/spack/core.repo
