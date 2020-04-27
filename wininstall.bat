@ECHO OFF

SET SHA_CMD="git rev-parse --short=10 HEAD"
FOR /F "tokens=* USEBACKQ" %%F IN (`%SHA_CMD%`) DO (
	SET SHA=%%F
)

SET HEAD_CMD="git rev-parse HEAD"
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

@ECHO ON
go get github.com/pauln/go-datefmt

:: go get github.com/gogo/protobuf/protoc-gen-gogo
go get -u github.com/gogo/protobuf/protoc-gen-gogo@21df5aa0e680850681b8643f0024f92d3b09930c
go get -u github.com/gogo/protobuf/protoc-gen-gogofaster@21df5aa0e680850681b8643f0024f92d3b09930c

:: go get github.com/gogo/protobuf/proto
go get -u github.com/gogo/protobuf/proto@21df5aa0e680850681b8643f0024f92d3b09930c

:: go get github.com/jteeuwen/go-bindata/...
go get -u github.com/kevinburke/go-bindata/go-bindata

go generate github.com/metaverse/truss/gengokit/template
go install -ldflags "-X 'main.Version=%SHA%' -X 'main.VersionDate=%HEAD_DATE%'" github.com/metaverse/truss/cmd/truss
@ECHO OFF
