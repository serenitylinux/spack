DEPS := $(shell find src/ -type f )
DEST := build
$(shell mkdir -p $(DEST))

export GOPATH=$(PWD)



all: $(DEST)/forge $(DEST)/wield $(DEST)/spack $(DEST)/smithy $(DEST)/spackle

$(DEST)/forge: forge.go $(DEPS)
	go build -o $(DEST)/forge forge.go
$(DEST)/spack: spack.go $(DEPS)
	go build -o $(DEST)/spack spack.go
$(DEST)/wield: wield.go $(DEPS)
	go build -o $(DEST)/wield wield.go
$(DEST)/smithy: smithy.go $(DEPS)
	go build -o $(DEST)/smithy smithy.go
$(DEST)/spackle: spackle.go $(DEPS)
	go build -o $(DEST)/spackle spackle.go

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

	install -c conf/* $(DESTDIR)/etc/spack/repos/

clean:
	rm $(DEST)/*
