package semver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var semverReg = regexp.MustCompile("^(\\d+).(\\d+).(\\d+)(?:-([0-9A-Za-z-.]+))?(?:\\+([0-9A-Za-z-.]+))?$")

type Semver struct {
	Semver     string `json:"semver"`
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	Prerelease string `json:"prerelease,omitempty"`
	Build      string `json:"build,omitempty"`
}

func Parse(semver string) (v Semver, err error) {
	pieces := semverReg.FindStringSubmatch(semver)
	if pieces == nil {
		err = fmt.Errorf("Invalid semver string: %s", semver)
		return
	}
	// will always be a number, but we're explicitly not checking for out of bounds errors
	v.Major, _ = strconv.Atoi(pieces[1])
	v.Minor, _ = strconv.Atoi(pieces[2])
	v.Patch, _ = strconv.Atoi(pieces[3])
	v.Prerelease = pieces[4]
	v.Build = pieces[5]
	v.Semver = semver
	err = v.Validate()
	return
}

func (v Semver) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

func (v *Semver) Validate() error {
	if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		return fmt.Errorf("Major, minor and patch version numbers must be non-negative")
	}
	return nil
}

func (ver *Semver) UnmarshalJSON(arr []byte) (err error) {
	var tmap map[string]interface{}
	if err = json.Unmarshal(arr, &tmap); err != nil {
		return
	}

	rVal := reflect.ValueOf(ver)
	for k, v := range tmap {
		field := rVal.Elem().FieldByName(strings.Title(k))
		valType := reflect.TypeOf(v)
		if valType.AssignableTo(field.Type()) {
			field.Set(reflect.ValueOf(v))
		} else if valType.ConvertibleTo(field.Type()) {
			field.Set(reflect.ValueOf(v).Convert(field.Type()))
		} else {
			// we'll only get here for Major, Minor & Patch
			if valType.Kind() == reflect.String {
				var val int
				val, err = strconv.Atoi(v.(string))
				if err != nil {
					return
				}
				field.SetInt(int64(val))
			}
		}
	}

	if ver.Semver == "" {
		return fmt.Errorf("semver must not be empty")
	}

	if ver.Major == 0 && ver.Minor == 0 && ver.Patch == 0 {
		*ver, err = Parse(ver.Semver)
	}

	if ver.String() != ver.Semver {
		return fmt.Errorf("semver must match parsed version")
	}

	return ver.Validate()
}

// Cmp compares two semantic versions:
// - < 0 if a < b
// - > 0 if a > b
// - == 0 if a == b
//
// In order of importance: Major > Minor > Patch > Prerelease (Build ignored)
//
// Major, Minor and Patch are compared numerically.
// Prerelease is compared by splitting on the . and:
// - comparing identifiers lexically (in ASCII sort order)
// - comparing numeric identifiers numerically
// Numeric identifiers have lower precedence
func (a Semver) Cmp(b Semver) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	if a.Patch != b.Patch {
		return a.Patch - b.Patch
	}

	if a.Prerelease == "" {
		if b.Prerelease == "" {
			return 0
		}
		return 1
	} else if b.Prerelease == "" {
		// a.Prerelease != ""
		return -1
	}

	partsA := strings.Split(a.Prerelease, ".")
	partsB := strings.Split(b.Prerelease, ".")
	total := len(partsA)
	if len(partsB) < total {
		total = len(partsB)
	}
	for i := 0; i < total; i++ {
		sa, sb := partsA[i], partsB[i]
		ai, errA := strconv.Atoi(sa)
		bi, errB := strconv.Atoi(sb)

		if errA != nil && errB != nil {
			if sa < sb {
				return -1
			} else if sa > sb {
				return 1
			}
		} else if errA == nil && errB == nil {
			if ai != bi {
				return ai - bi
			}
		} else if errA != nil {
			return 1
		} else if errB != nil {
			return -1
		}
	}

	return len(partsA) - len(partsB)
}
