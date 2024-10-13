pfx1=_386.go
pfx2=_amd64.go

f1=$1$pfx1
f2=$1$pfx2

sed '/^\/\/ mksyscall.*\.pl/d' <$f1 >,,u1
sed '/^\/\/ mksyscall.*\.pl/d' <$f2 >,,u2

if cmp -s ,,u1 ,,u2; then
	echo unifying $f1 and $f2
	mv $f1 $1.go
	rm $f2
fi

rm -f ,,u[12]
