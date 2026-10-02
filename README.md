## Cueform

> XLSForm is a form standard created to help simplify the authoring of forms in Excel. Authoring is done in a human-readable format using a familiar tool that almost everyone knows - Excel

Source: [XLSForm website](https://xlsform.org/en/)

This repo provides a tool that converts CUE to XLSForm. It has full compatibility with XLS forms so we can do both `CUE -> XLSForm` and `XLSForm -> CUE` conversions. It uses the [same spec](https://xlsform.org/en/ref-table/) used by XLS forms. You can find example forms in the examples directory.

### Download and Install

#### Install from source

    go install github.com/freddieptf/cueform/cmd/cueform@latest

You also need the [`cue` command](https://cuelang.org/docs/introduction/installation/), v0.16 or
newer, to manage the schema dependency.

#### Usage

    ./cueform --help

### Getting started

Forms are CUE files that import cueform's schema, `github.com/freddieptf/cueform/xlsform`. The
schema is published to the [CUE Central Registry](https://registry.cue.works) in the
`github.com/freddieptf/cueform@v0` module, so the usual CUE tooling fetches it.

1. Create a CUE module for your forms:

       mkdir myforms && cd myforms
       cue mod init example.com/myforms@v0

2. Write a form, for example `survey.cue`:

       package myforms

       import "github.com/freddieptf/cueform/xlsform"

       name: xlsform.#Question & {
           type:  "text"
           name:  "name"
           label: "What is your name?"
       }

       form_settings: xlsform.#Settings & {
           type:       "settings"
           form_title: "My survey"
           form_id:    "my_survey"
       }

3. Add the schema to the module's dependencies:

       cue mod tidy

4. Check the form with CUE, and encode it to an XLSForm:

       cue vet .
       cueform encode survey.cue

   `cueform encode` writes `survey.xlsx` next to where you run it. Add `-out <dir>` to write it
   elsewhere.

To start from an existing XLSForm instead, decode it inside your module and tidy:

    cueform decode survey.xlsx
    cue mod tidy

The decoded `survey.cue` imports the schema, so it works the same way.

Dependencies come from the Central Registry by default. Set `CUE_REGISTRY` to use another
registry. To move to a newer schema, run `cue mod get github.com/freddieptf/cueform@v0.X.Y`, and
use a cueform release of the same version: `cueform encode` validates against the schema built
into the binary.

`cueform yank labels` moves a form's translations into a separate `labels.cue`; see
[examples/yank](examples/yank).
