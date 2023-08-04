cd $1
gen() {
	GOOS=linux GOARCH=$1 sh genarch.sh
}

gen arm
gen arm64
gen 386
gen amd64
