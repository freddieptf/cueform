### Yank

`cueform yank labels` moves a form's translated text out of the form and into a `labels.cue`
file next to it. The form keeps its structure, and each translation becomes a reference such as
`_labels."household/head_age/label"`. Translators can then work in one file without touching
the form.

`before/form.cue` is a form as you would write it. `after/` is the same form after

    cd before
    cueform yank labels form.cue

which rewrites `form.cue` in place and writes `labels.cue`. Add `-dry` to print both files instead
of writing them. Running it again on a yanked form changes nothing.

#### What the example shows

- **Translated values move to `labels.cue`.** Each `{"English (en)": ..., "Swahili (sw)": ...}`
  label and hint is replaced by a reference. Entries are keyed by the element's path and the
  column (`"household/head_age/hint"`), so questions with the same name in different groups
  get different entries. Choices are keyed by list and choice (`"water_sources/tap"`). If a key
  is already taken by other text, for example in an existing `labels.cue`, it gets a numbered
  suffix such as `-2`.
- **Repeated text is stored once.** `member_age` has the same translations as `head_age`, so
  both refer to `_labels."household/head_age/label"`. Values share an entry only when every language
  matches, and entries already in `labels.cue` are reused when you run yank again.
- **A choice with media keeps its media.** For `{well: {label: ..., image: "well.png"}}`, only
  the label moves.
- **Single-language values stay put.** The `intro` note's plain `label` isn't a translation,
  so it isn't yanked.

The encoder loads `labels.cue` with the form, so both versions encode to the same spreadsheet:

    cueform encode -out /tmp examples/yank/before/form.cue
    cueform encode -out /tmp examples/yank/after/form.cue

Only struct literals are yanked, wherever they are: `#Question & {...}`, a plain `{...}`, an
element inside `list.Concat`, or a helper such as `_tree_name: #Question & {...}` or
`_yes_no: #Choices & {...}` that the form refers to by name. References themselves are left
alone, so questions imported from another package keep their translations there.

#### Current limits

- `labels.cue` is always `package main`, so the form must be `package main` too.
- Translated media, such as `image: {"English (en)": "a.png"}`, is yanked along with the text.
