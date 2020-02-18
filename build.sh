#PLATFORM=`uname -s | tr '[:upper:]' '[:lower:]'`-`uname -m`
set -ex
export GOOS=darwin
export GOARCH=amd64
mkdir -p dist/$TRAVIS_BUILD_NUMBER/$GOOS-$GOARCH
go build -o dist/$TRAVIS_BUILD_NUMBER/$GOOS-$GOARCH/conseq main.go
export GOOS=linux
export GOARCH=amd64
mkdir -p dist/$TRAVIS_BUILD_NUMBER/$GOOS-$GOARCH
go build -o dist/$TRAVIS_BUILD_NUMBER/$GOOS-$GOARCH/conseq main.go
