#PLATFORM=`uname -s | tr '[:upper:]' '[:lower:]'`-`uname -m`
mkdir -p dist
env GOOS=darwin GOARCH=amd64 go build package-import-path
go build -o dist/conseq-$GOOS-$GOARCH main.go
env GOOS=linux GOARCH=amd64 go build package-import-path
go build -o dist/conseq-$GOOS-$GOARCH main.go
