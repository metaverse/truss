package svcdef

// typeBox holds either a Message or an Enum; used only in resolveTypes() to
// associate FieldTypes with their underlying data.
type typeBox struct {
	Message *Message
	Enum    *Enum
}

// newTypeMap returns a map of type (Message/Enum) Names to correlated
// typeBoxes
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

// resolveTypes sets the underlying types for all of our service methods
// Fields, RequestTypes, and ResponseTypes. Since Enums have no Fields which
// may refer to other types, they are ignored by resolveTypes.
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

// setType unpacks typeBox value into corresponding FieldType
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
