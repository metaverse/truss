@ECHO OFF
REM In order to install Truss, you need to have:
REM - Go installed
REM - GOPATH configured
REM - Glide installed: See: http://glide.sh/

REM Install all dependencies
glide install

REM install 3rd-party binaries
echo Installing required 3rd-party binaries...
go get github.com/golang/protobuf/protoc-gen-go
go get github.com/jteeuwen/go-bindata/...
go get github.com/pauln/go-datefmt
echo done!

SET SEMVER_CMD="type VERSION"
FOR /F "tokens=* USEBACKQ" %%F IN (`%SEMVER_CMD%`) DO (
    SET SEMVER=%%F
)

SET HEAD_CMD="git rev-list -n 1 %SEMVER%"
FOR /F "tokens=* USEBACKQ" %%F IN (`%HEAD_CMD%`) DO (
	SET HEAD_COMMIT=%%F
)

SET HEAD_DATE_CMD="git show -s --format=%%ct %HEAD_COMMIT%"
FOR /F "tokens=* USEBACKQ" %%F IN (`%HEAD_DATE_CMD%`) DO (
	SET GIT_COMMIT_EPOC=%%F
)

SET DATE_FMT_CMD="go-datefmt -ts %GIT_COMMIT_EPOC% -fmt UnixDate -utc"
FOR /F "tokens=* USEBACKQ" %%F IN (`%DATE_FMT_CMD%`) DO (
	SET HEAD_DATE=%%F
)

echo Generating templates...
go generate github.com/TuneLab/truss/gengokit/template
echo building truss...
go install github.com/TuneLab/truss/cmd/protoc-gen-truss-protocast
go install -ldflags "-X 'main.Version=%SEMVER%' -X 'main.VersionDate=%HEAD_DATE%'" github.com/TuneLab/truss/cmd/truss && echo success! && EXIT /B 0
echo oh no! Check for failures...
