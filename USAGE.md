# Using Truss

## File structure

Start with your definition file, named, for example, `svc.proto`, in the
root of your project:

```
.
└── svc.proto
```

Invoke Truss on your service definition:
```
$ truss svc.proto
```

After running truss on your service, two new directories should be created in
your current directory. One is called `third_party`, and the other is called
`NAME-service`, where NAME is the name of the package defined in your
definition file. The directory tree after running truss will look like this:

```
.
├── NAME-service
│   ├── docs
│   │   └── docs.md
│   ├── generated
│   │   └── ...
│   ├── handlers
│   │   ├── client
│   │   │   └── client_handler.go
│   │   └── server
│   │       └── server_handler.go
│   ├── NAME-client
│   │   └── client_main.go
│   ├── NAME-server
│   │   └── server_main.go
│   └── svc.pb.go
├── svc.proto
└── third_party
    └── ...
```

Now that you've generated your service, you can install the generated binaries
with `go install ./...` which will install `NAME-client` and `NAME-server`,
where NAME is the name of the package defined in your definition file. Of
course, these binaries won't do anything useful until you add business logic.

To add business logic from this point, you'd edit the `server_handler.go` file
in `./NAME-service/handlers/server/`, where NAME is the name of the package
defined in your definition file.

## *Our Contract*

1. Only the files in `handlers/` are user modifiable. The only functions
   allowed in the handler files are functions with the same names as those
   defined as RPCs in the definition service, as well as a `NewService`
   function used to create an instance of a service; all other functions will
   be removed. User logic can be imported and executed within the functions in
   the handlers.
2. Do not create files or directories in `NAME-service/`
3. All user logic should exist outside of `NAME-service/`, leaving organization
   of that logic up to the user.
