DEPS := $(shell find ../libspack/ -type f )
DEST := build
$(shell mkdir -p $(DEST))

all: $(DEST)/forge $(DEST)/wield $(DEST)/spack $(DEST)/smithy $(DEST)/spackle

$(DEST)/forge: forge/forge.go $(DEPS)
	go build -o $(DEST)/forge forge/forge.go
$(DEST)/spack: spack/spack.go $(DEPS)
	go build -o $(DEST)/spack spack/spack.go
$(DEST)/wield: wield/wield.go $(DEPS)
	go build -o $(DEST)/wield wield/wield.go
$(DEST)/smithy: smithy/smithy.go $(DEPS)
	go build -o $(DEST)/smithy smithy/smithy.go
$(DEST)/spackle: spackle/spackle.go $(DEPS)
	go build -o $(DEST)/spackle spackle/spackle.go

install:
	mkdir -p $(DESTDIR)/var/lib/spack
	mkdir -p $(DESTDIR)/var/cache/spack
	mkdir -p $(DESTDIR)/etc/spack/repos
	mkdir -p $(DESTDIR)/usr/bin/

	install -c $(DEST)/forge  $(DESTDIR)/usr/bin/forge
	install -c $(DEST)/wield  $(DESTDIR)/usr/bin/wield
	install -c $(DEST)/spack  $(DESTDIR)/usr/bin/spack
	install -c $(DEST)/smithy $(DESTDIR)/usr/bin/smithy
	install -c $(DEST)/spackle  $(DESTDIR)/usr/bin/spackle

	install -c conf/*.conf $(DESTDIR)/etc/spack/repos/
	install -c conf/*.sh $(DESTDIR)/etc/spack/

clean:
	rm $(DEST)/*
