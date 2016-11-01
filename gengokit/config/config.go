package config

import "github.com/TuneLab/go-truss/truss/truss"

type Config struct {
	GoPackage string
	PBPackage string

	PreviousFiles []truss.NamedReadWriter
}
