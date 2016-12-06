# Using truss

## File structure

Start with your service definition file, let's name it `svc.proto`. 
For a more detailed example of a simple service definition see `go-truss/example/echo.proto`,
but for now we only care about the following structure:
```
// The package name determines the name of the directories that truss creates
package NAME;

// RPC interface definitions
...
```

The current directory should look like this:

```
.
└── svc.proto
```

Run truss on your service definition file: `truss svc.proto`.  
Upon success, `NAME-service` folder will be created in your current directory. 
(`NAME-service`, where NAME is the name of the package defined in your definition file.)

Your directory structure will look like this:

```
.
├── NAME-service
│   ├── docs
│   │   └── docs.md
│   ├── generated
│   │   └── ...
│   ├── handlers
│   │   └── server
│   │       └── server_handler.go
│   ├── middlewares
│   │   └── ...
│   ├── NAME-cli-client
│   │   └── client_main.go
│   ├── NAME-server
│   │   └── server_main.go
│   └── svc.pb.go
├── svc.proto
```

Now that you've generated your service, you can install the generated binaries
with `go install ./...` which will install `NAME-cli-client` and `NAME-server`,
where NAME is the name of the package defined in your definition file.

To add business logic, edit the `server_handler.go` file in `./NAME-service/handlers/server/`.

To add middlewares, edit ... (TODO)

## Our Contract

1. Modify ONLY the files in `handlers/` and `middlewares/`.

 User logic can be imported and executed within the functions in the handlers. It can also be added as _unexported_ helper functions in the handler file. 

 Truss will enforce that exported functions in `server_handler.go` conform to the rpc interface defined in the service *.proto files. All other exported functions will be removed upon next re-run of truss. 

2. DO NOT create files or directories in `NAME-service/`
 All user logic must exist outside of `NAME-service/`, leaving organization of that logic up to the user.
