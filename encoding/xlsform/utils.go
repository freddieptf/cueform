package xlsform

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
)

func indexOf[K interface{}](arr []K, val K) int {
	for idx, item := range arr {
		if reflect.DeepEqual(item, val) {
			return idx
		}
	}
	return -1
}

var (
	moduleRe = regexp.MustCompile(`module:\s+"(.+)"`)
)

func getModuleName(path string) (string, error) {
	moduleCueFile := filepath.Join(path, "cue.mod", "module.cue")
	contentBytes, err := ioutil.ReadFile(moduleCueFile)
	if err != nil {
		return "", fmt.Errorf("%w\ndid you forget to pass a module? run --help to see usage", err)
	}
	matches := moduleRe.FindStringSubmatch(string(contentBytes[:]))
	if len(matches) < 2 {
		return "", fmt.Errorf("couldn't find module name in %s", moduleCueFile)
	}
	return matches[1], nil
}

func loadFile(module, path string) (*cue.Value, error) {
	ctx := cuecontext.New()
	bis := load.Instances([]string{path}, &load.Config{ModuleRoot: module})
	bi := bis[0]
	// check for errors on the instance
	// these are typically parsing errors
	if bi.Err != nil {
		return nil, fmt.Errorf("Error during load: %w", bi.Err)
	}
	// Use cue.Context.BuildInstance to turn
	// a build.Instance into a cue.Value
	value := ctx.BuildInstance(bi)
	if value.Err() != nil {
		return nil, fmt.Errorf("Error during build: %w", value.Err())
	}
	// Validate the value
	err := value.Validate(cue.Concrete(true))
	if err != nil {
		return nil, fmt.Errorf("Error during validation: %v", errors.Details(err, nil))
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
