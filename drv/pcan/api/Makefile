P=api

W=windows
L=linux

ZFILESWIN=\
	ztypes_$W_386.go\
	z$P_$W_386.go\
	ztypes_$W_amd64.go\
	z$P_$W_amd64.go\


ZFILESLIN=\
	ztypes_$L_386.go\
	z$P_$L_386.go\
	ztypes_$L_amd64.go\
	z$P_$L_amd64.go\

LINAPIFILES=	\
	zmethods_linux.go\
	,,lintypes.go\

all:	zfiles unify

zfiles: \
	$(ZFILESWIN)\
	zdefs_$W.go\
	zdefs.go\
	\
	$(ZFILESLIN)\

unify:
	for f in zapi_linux ztypes_linux zapi_windows ztypes_windows; do \
		sh unify.sh $$f; \
	done \

zdef%.go: \
	PCANBasic.h\
	$W/defs.sh\
	$W/defs.awk\
	$W/param-map\

	sh ./$W/defs.sh $* > $@

$(ZFILESWIN): \
	$W/gen.sh\
	$W/types.go\
	$P_$W.go\

	GOARCH=386 GOOS=$W sh ./$W/gen.sh
	GOARCH=amd64 GOOS=$W sh ./$W/gen.sh

$(ZFILESLIN): $(LINAPIFILES)\
	$L/gen.sh\
	,,lintypes.go\
	$P_$L.go\

	GOARCH=386 GOOS=$L sh ./$L/gen.sh
	GOARCH=amd64 GOOS=$L sh ./$L/gen.sh

$(LINAPIFILES):	./$L/api-defs.sh
	sh $<

clean:
	rm -rf _obj
	rm -f ,,*

nuke: clean
	rm -f z*.go


.PHONY: all clean nuke
