tool=x86_64-w64-mingw32-

all: jpcangohelper.dll

LIBS=../jpcan.dll

%.dll: %.o
	$(tool)gcc -shared -o $@ $< $(LIBS)

%.o: ../%.c
	$(tool)gcc -c $<
