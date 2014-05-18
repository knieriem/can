P=pcan

W=windows
L=linux

ZFILESWIN=\
	z$P_$W_386.go\
	z$P_$W_amd64.go\


all:	zfiles unify

zfiles: \
	$(ZFILESWIN)\
	\
	$(ZFILESLIN)\

unify:
	for f in zpcan_windows; do \
		sh ../../pcan/api/unify.sh $$f; \
	done \

$(ZFILESWIN): \
	$W/gen.sh\
	$P_$W.go\

	GOARCH=386 GOOS=$W sh ./$W/gen.sh
	GOARCH=amd64 GOOS=$W sh ./$W/gen.sh

clean:
	rm -rf _obj
	rm -f ,,*

nuke: clean
	rm -f z*.go


.PHONY: all clean nuke
