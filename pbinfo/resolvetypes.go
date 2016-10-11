package pbinfo

// After the initial creation of Catalog, each FieldType will have it's name
// set to the name of the type that it represents, but it won't have a pointer
// to the full instance of the type with that name. Type resolution is the
// process of taking the name of each FieldType, then searching for the
// enum/message with that name and setting the respective *Message/*Enum
// property of our FieldType to point to that enum/message.

type typeBox struct {
	Message *Message
	Enum    *Enum
}

// resolveTypes accepts a pointer to a Catalog and modifies that Catalog, and
// it's child structs, in place.
func resolveTypes(cat *Catalog) {
	tmap := newTypeMap(cat)
	for _, m := range cat.Messages {
		for _, f := range m.Fields {
			setType(f.Type, tmap)
		}
	}
	if cat.Service != nil {
		for _, m := range cat.Service.Methods {
			setType(m.RequestType, tmap)
			setType(m.ResponseType, tmap)
		}
	}
}

func setType(f *FieldType, tmap map[string]typeBox) {
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
func newTypeMap(cat *Catalog) map[string]typeBox {
	rv := make(map[string]typeBox)
	for _, m := range cat.Messages {
		rv[m.Name] = typeBox{Message: m}
	}
	for _, e := range cat.Enums {
		rv[e.Name] = typeBox{Enum: e}
	}
	return rv
}
