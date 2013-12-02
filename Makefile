all:

install:
	mkdir -p $(DESTDIR)/lib/spack
	mkdir -p $(DESTDIR)/bin
	install -c src/forge/forge.sh $(DESTDIR)/bin/forge
	install -c src/wield/wield.sh $(DESTDIR)/bin/wield
	install -c src/spack/spack.sh $(DESTDIR)/bin/spack
	install -c src/libspack.sh    $(DESTDIR)/lib/spack/libspack
