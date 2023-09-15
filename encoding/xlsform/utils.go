package xlsform

import (
	"fmt"
	"reflect"

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

func loadFile(path string) (*cue.Value, error) {
	ctx := cuecontext.New()
	bis := load.Instances([]string{path}, &load.Config{ModuleRoot: ""})
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
