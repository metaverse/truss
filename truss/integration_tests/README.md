# integration_tests

The truss integration runner does the following tasks:
- Runs truss on each service definition in `test_service_definitions`
- Builds the server and cliclient for each service
- Runs the server
- Runs the cliclient against the server
- Passes if the server and cliclient were able to communicate. Fails if there were errors of any kind.

## Test service definition requirements

Each service definition must have the package name `TEST`, all letters uppercase.
