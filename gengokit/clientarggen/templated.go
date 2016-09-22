package clientarggen

import (
	"fmt"
	"strconv"
	"strings"
)

//

func CarveIntegerthirtytwo(inpt string) []int32 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []int32

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseInt(item, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, int32(tmp))
	}
	return rv
}

func CarveIntegersixtyfour(inpt string) []int64 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []int64

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, tmp)
	}
	return rv
}

func CarveUintthirtytwo(inpt string) []uint32 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []uint32

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseUint(item, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, uint32(tmp))
	}
	return rv
}

func CarveUintsixtyfour(inpt string) []uint64 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []uint64

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, tmp)
	}
	return rv
}

func CarveFloatthirtytwo(inpt string) []float32 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []float32

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseFloat(item, 32)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, float32(tmp))
	}
	return rv
}

func CarveFloatsixtyfour(inpt string) []float64 {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []float64

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseFloat(item, 64)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, tmp)
	}
	return rv
}

func CarveBool(inpt string) []bool {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []bool

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.ParseBool(item)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, tmp)
	}
	return rv
}

func CarveString(inpt string) []string {
	inpt = strings.Trim(inpt, "[] ")
	slc := strings.Split(inpt, ",")
	var rv []string

	for _, item := range slc {
		item = strings.Trim(item, " ")
		item = strings.Replace(item, "'", "\"", -1)
		if len(item) == 0 {
			continue
		}
		tmp, err := strconv.Unquote(item)
		if err != nil {
			panic(fmt.Sprintf("couldn't parse '%v' of '%v'", item, inpt))
		}
		rv = append(rv, tmp)
	}
	return rv
}
