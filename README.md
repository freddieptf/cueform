## Cueform

> XLSForm is a form standard created to help simplify the authoring of forms in Excel. Authoring is done in a human-readable format using a familiar tool that almost everyone knows - Excel

Source: [XLSForm website](https://xlsform.org/en/)

This repo provides a tool that converts CUE to XLSForm. It has full compatibility with XLS forms so we can do both `CUE -> XLSForm` and `XLSForm -> CUE` conversions. It uses the [same spec](https://xlsform.org/en/ref-table/) used by XLS forms. You can find example forms in the sample directory.

### Download and Install

#### Install from source

    go install github.com/freddieptf/cueform/cmd/cue2xlsform@latest

#### Usage

    ./cue2xlsform --help
