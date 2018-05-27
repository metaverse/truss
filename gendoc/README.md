## NOTE: Unused.
## Hopefully in the future we can use something like swagger to generate docs

# `gendocs`

A `truss` plugin which can generate documentation from an annotated Protobuf definition file. Handles http-options.

## Limitations and Bugs

Currently, there are a variety of limitations in the documentation parser.

- Having additional http bindings via the `additional_bindings` directive when declaring http options causes the parser to break.
