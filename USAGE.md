# Using truss
## Our Contract

1. Modify ONLY the files in `handlers/`.

 User logic can be imported and executed within the functions in the handlers. It can also be added as _unexported_ helper functions in the handler file.

 Truss will enforce that exported functions in `handlers/handlers.go` conform to the rpc interface defined in the service *.proto files. All other exported functions will be removed upon next re-run of truss.

2. DO NOT create files or directories in `NAME-service/`
 All user logic must exist outside of `NAME-service/`, leaving organization of that logic up to the user.
