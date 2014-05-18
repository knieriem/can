tool=i686-w64-mingw32-
DLL=jpcangohelper.dll

all: $(DLL)

LIBS=../jpcan.dll

%.dll: %.o
	$(tool)gcc -shared -o $@ $< $(LIBS)

%.o: ../%.c
	$(tool)gcc -O2 -Wall -c $<

clean:
	rm -f $(DLL)
