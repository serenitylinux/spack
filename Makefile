DEPS := $(shell find src/libspack/ -type f )

export GOPATH=$(shell pwd)

all: forge wield spack

forge: forge.go ${DEPS}
	go build forge.go
spack: spack.go ${DEPS}
	go build spack.go
wield: wield.go ${DEPS}
	go build wield.go

install:
	mkdir -p $(DESTDIR)/var/lib/spack
	mkdir -p $(DESTDIR)/var/cache/spack
	mkdir -p $(DESTDIR)/etc/spack/repos

	install -c forge $(DESTDIR)/usr/bin/forge
	install -c wield $(DESTDIR)/usr/bin/wield
	install -c spack $(DESTDIR)/usr/bin/spack

	install -c conf/core.repo $(DESTDIR)/etc/spack/repos/core.repo

clean:
	rm forge spack wield
