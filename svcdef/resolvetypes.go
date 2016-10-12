package svcdef

// After the initial creation of Svcdef, each FieldType will have it's name
// set to the name of the type that it represents, but it won't have a pointer
// to the full instance of the type with that name. Type resolution is the
// process of taking the name of each FieldType, then searching for the
// enum/message with that name and setting the respective *Message/*Enum
// property of our FieldType to point to that enum/message.

type typeBox struct {
	Message *Message
	Enum    *Enum
}

// resolveTypes accepts a pointer to a Svcdef and modifies that Svcdef, and
// it's child structs, in place.
func resolveTypes(sd *Svcdef) {
	tmap := newTypeMap(sd)
	for _, m := range sd.Messages {
		for _, f := range m.Fields {
			setType(f.Type, tmap)
		}
	}
	if sd.Service != nil {
		for _, m := range sd.Service.Methods {
			setType(m.RequestType, tmap)
			setType(m.ResponseType, tmap)
		}
	}
}

func setType(f *FieldType, tmap map[string]typeBox) {
	// Special case maps with valuetypes pointing to messages
	if f.Map != nil {
		if f.Map.ValueType.StarExpr {
			f = f.Map.ValueType
		}
	}
	entry, ok := tmap[f.Name]
	if !ok {
		return
	}
	switch {
	case entry.Enum != nil:
		f.Enum = entry.Enum
	case entry.Message != nil:
		f.Message = entry.Message
	}
}

// newTypeMap returns a map from
func newTypeMap(sd *Svcdef) map[string]typeBox {
	rv := make(map[string]typeBox)
	for _, m := range sd.Messages {
		rv[m.Name] = typeBox{Message: m}
	}
	for _, e := range sd.Enums {
		rv[e.Name] = typeBox{Enum: e}
	}
	return rv
}
