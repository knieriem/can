#include <windows.h>

#define JPCAN __declspec(dllimport)
#define API __declspec(dllexport)
typedef unsigned char uint8;
#define nil NULL
#define	nelem(x)	(sizeof(x)/sizeof((x)[0]))

extern	long	JPCAN pcan_register_callback(int fd, void *callback);
extern	long	JPCAN pcan_deregister_callback(int fd);
extern	long	JPCAN pcan_list_usb_devices(uint8**, uint8*);

extern	int	API	create(int fd);
extern	int	API	readmsg(int fd, uint8*);
extern	int	API	msgavail(int fd);
extern	void	API	close(int fd);
extern	int	API	usbdevices(uint8 *, int n);


enum {
	InitialOff = 0,
	ManualReset = 1,
};

typedef struct Device Device;

struct Device {
	int fd;
	HANDLE evavail;
	HANDLE evdone;
	uint8 *data;
};

static Device dev[16];
static int ndev;

static Device*
getdev(int fd)
{
	int i, n;

	n = ndev;

	for (i=0; i<n; i++) {
		if (dev[i].fd == fd)
			return &dev[i];
	}
	if (n < nelem(dev)) {
		dev[i].fd = fd;
		ndev++;
		return &dev[i];
	}
	return nil;
}

static void
cb(int fd, uint8 *p)
{
	Device *d;

	d = getdev(fd);
	if (d==nil)
		return;

	d->data = p;
	SetEvent(d->evavail);

	WaitForSingleObject(d->evdone, INFINITE);
	ResetEvent(d->evdone);

}

int
create(int fd)
{
	Device *d;
	long st;

	d = getdev(fd);
	if (d == nil)
		return -1;
	d->evavail = CreateEventA(0, ManualReset, InitialOff, NULL);
	if (d->evavail == INVALID_HANDLE_VALUE)
		return -1;
	d->evdone = CreateEventA(0, ManualReset, InitialOff, NULL);
	if (d->evdone == INVALID_HANDLE_VALUE)
		return -1;

	st = pcan_register_callback(fd, cb);
	return st;
}

int
readmsg(int fd, uint8 *buf)
{
	DWORD ret;
	Device *d;
	int i;

	d = getdev(fd);
	if (d == nil)
		return -1;
	ret = WaitForSingleObject(d->evavail, INFINITE);
	if (ret == WAIT_FAILED)
		return -1;
	ResetEvent(d->evavail);
	for (i=0; i<32; i++) {
		buf[i] = d->data[i];
	}
	SetEvent(d->evdone);
	return 0;	
}

int
msgavail(int fd)
{
	Device *d;

	d = getdev(fd);
	if (d == nil)
		return 0;
	return WaitForSingleObject(d->evavail, 0) != WAIT_TIMEOUT;

}

void
close(int fd)
{
	Device *d;

	d = getdev(fd);
	if (d == nil) {
		return;
	}
	pcan_deregister_callback(fd);
	CloseHandle(d->evavail);
	CloseHandle(d->evdone);
	d->fd = -1;
}

enum {
	sizeofDevInfo = 32+32+4*16+64+4
};

int
usbdevices(uint8 *buf, int sz)
{
	uint8 *p, n;
	int i, np;

	if (pcan_list_usb_devices(&p, &n) <0)
		return -1;
	if (n > sz)
		n = sz;

	np = (int)n*sizeofDevInfo;
	for (i=0; i<np; i++) {
		buf[i] = p[i];
	}
	free(p);
	return n;
}
