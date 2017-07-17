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
go get github.com/golang/protobuf/protoc-gen-go
go get github.com/golang/protobuf/proto
go get github.com/jteeuwen/go-bindata/...
go generate github.com/TuneLab/truss/gengokit/template
go install github.com/TuneLab/truss/cmd/protoc-gen-truss-protocast
go install -ldflags "-X 'main.Version=%SHA%' -X 'main.VersionDate=%HEAD_DATE%'" github.com/TuneLab/truss/cmd/truss
@ECHO OFF