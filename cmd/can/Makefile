RESOFILES=\
	res_windows_amd64.syso\
	res_windows_386.syso\

all:	resource-objects

resource-objects: $(RESOFILES)

dist:
	rc ./mkdist.rc

clean:
	rm -f $(RESOFILES)
	GOOS=windows go clean
	GOOS=linux go clean

PUBDIR=$(home)/ib/wmipf.de/home/www/ib/tools/can
publish:
	cp can-*tar.gz $(PUBDIR)

res_windows_%.syso:
	GOOS=windows GOARCH=$* sh windows/windres.sh $@

.PHONY: all clean resource-objects
