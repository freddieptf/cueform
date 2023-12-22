package xlsform

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
)

func IsTranslatableColumn(column string) bool {
	for _, item := range TranslatableCols {
		if strings.HasPrefix(column, item) {
			return true
		}
	}
	return false
}

func GetLangFromCol(translatableColumn string) (col string, lang string, err error) {
	match := langRe.FindStringSubmatch(translatableColumn)
	if len(match) != 3 || slices.Index(TranslatableCols, match[1]) == -1 {
		err = fmt.Errorf("missing lang code %s: %w", translatableColumn, ErrInvalidLabel)
		return
	}
	col = match[1]
	lang = match[2]
	return
}

func LoadInstance(path string) ([]*build.Instance, error) {
	formPaths := []string{path}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "labels.cue")); err == nil {
		formPaths = append(formPaths, filepath.Join(filepath.Dir(path), "labels.cue"))
	}
	bis := load.Instances(formPaths, &load.Config{ModuleRoot: ""})
	if bis[0].Err != nil {
		return nil, fmt.Errorf("error during load: %s", errors.Details(bis[0].Err, nil))
	}
	return bis, nil
}

func LoadValue(path string) (*cue.Value, error) {
	bis, err := LoadInstance(path)
	if err != nil {
		return nil, err
	}
	ctx := cuecontext.New()
	value := ctx.BuildInstance(bis[0])
	if value.Err() != nil {
		return nil, fmt.Errorf("error during build: %s", errors.Details(value.Err(), nil))
	}
	err = value.Validate(cue.Concrete(true))
	if err != nil {
		return nil, fmt.Errorf("error during validation: %s", errors.Details(err, nil))
	}
	return &value, nil
}

func getIter(val *cue.Value) (*cue.Iterator, error) {
	switch val.Eval().Kind() {
	case cue.StructKind:
		if iter, err := val.Fields(cue.Concrete(true)); err != nil {
			return nil, err
		} else {
			return iter, nil
		}
	case cue.ListKind:
		if iter, err := val.List(); err != nil {
			return nil, err
		} else {
			return &iter, nil
		}
	default:
		return nil, fmt.Errorf("no %+v", val)
	}
}

func getHeadersInOrder(headers map[string]struct{}, parentList []string) []string {
	columnHeaders := []string{}
	for _, header := range parentList {
		if _, ok := headers[header]; ok {
			columnHeaders = append(columnHeaders, header)
			delete(headers, header)
		} else {
			cols := []string{}
			for key := range headers {
				match := langRe.FindStringSubmatch(key)
				if len(match) != 3 {
					continue
				}
				if header == match[1] {
					cols = append(cols, key)
					delete(headers, key)
				}
			}
			sort.Strings(cols)
			columnHeaders = append(columnHeaders, cols...)
		}
	}
	moarFields := []string{}
	for key := range headers {
		moarFields = append(moarFields, key)
	}
	sort.Strings(moarFields)
	columnHeaders = append(columnHeaders, moarFields...)
	return columnHeaders
}
