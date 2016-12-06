# Tutorial
We will build a simple service based on [echo.proto](./example/echo.proto)

# Installation tips
1. Follow instructions in the [README](./README.md)
  - Can use `brew install` to get protobuf, golang, but not other packages
2. Go to truss installation folder and run `make test`
  If everything passes you’re good to go.
  If you see any complaints about packages not installed, `go get` those packages
  If you encounter any other issues - ask the developers
3. To update to newer version of truss, do `git pull`, or `go get -u github.com/TuneLab/go-truss/...` truss again.

# Writing your first service
Define the communication interface for your service in the *.proto file(s). 
Start with [echo.proto](./example/echo.proto) and read the helpful comments.

## What is in the *.proto file definitions? 


## Understanding generated structures

## Implement business logic

## Build/Run the client and server executables


# Additional features

## File placement
You can control the location of the output folders for your service by specifying the following flags when running truss
```
  -svcout {go-style-package-path to where you want the {Name}-service folder to be}
  -pbout {go-style-package-path to where you want the *.pb.go interface definitions to be}
```

Note: “go-style-package-path” means exactly the style you use in your golang import statements, relative to your $GOPATH. This is not your system file path, nor it is relative to location of the *.proto file; the start of the path must be accessible from your $GOPATH. Also no “/” at the end. 
For example:
```
truss -pbout truss-demo/interface-defs -svcout truss-demo/service echo.proto
```
Executing this command will place the *.pb.go files into `$GOPATH/truss-demo/interface-defs/`, and the entire echo-service directory (excepting the *.pb.go files) to `$GOPATH/truss-demo/service/`.

## Middlewares
 TODO