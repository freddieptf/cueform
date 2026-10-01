"""Fixtures for checking that the decoder reads cells the way pyxform does.

    python cells.py write        build cells.xlsx and record pyxform's reading of it in cells.json
    python cells.py read FILE    print pyxform's reading of FILE's survey "default" column as JSON

Needs pyxform==4.5.0, which pins openpyxl==3.1.5.
"""

import datetime
import json
import os
import sys

from openpyxl import Workbook
from pyxform.xls2json_backends import xlsx_to_dict

HERE = os.path.dirname(os.path.abspath(__file__))

# (value, number format) for the kinds of value XLSForm columns hold. A format of None keeps the
# one openpyxl picks for the value.
CELLS = [
    # numbers (repeat_count, default, version, choice names): the stored value, not the display
    (3, "0.00"),
    (0.5, "0.00%"),
    (1234.5, "#,##0.000"),
    (3.25, '0.00 "kg"'),
    (-1.5, "#,##0.00_);[Red](#,##0.00)"),
    (0.000123, "0.00"),
    (2500.0, "0.0E+00"),
    (45931, "[$-409]0.00"),
    (7, "0"),
    (1, "General"),
    (2026100101, "General"),
    (1234567.5, "General"),
    (0.1 + 0.2, "General"),
    # text and bools read as they are
    ("3.00", "General"),
    ("007", "General"),
    ("2024-01-15", "@"),
    (True, "General"),
    (False, "General"),
    # date cells, which Excel makes from typed text: a date or time default, or a hint typed as 1/2
    (datetime.date(2024, 1, 15), None),
    (datetime.datetime(2025, 10, 1, 8, 30), None),
    (datetime.time(13, 30), None),
    (datetime.datetime(2026, 1, 2), "d-mmm"),
    (45931, "mm-dd-yy"),
    (45931, "[$-409]d-mmm-yy"),
    (45931.5, "yyyy-mm-dd hh:mm"),
    (0.75, "h:mm"),
    (0.5, "h:mm:ss AM/PM"),
]


def build(path):
    wb = Workbook()
    ws = wb.active
    ws.title = "survey"
    ws.append(["type", "name", "label::en", "default"])
    for i, (value, number_format) in enumerate(CELLS):
        ws.append(["text", f"q{i}", f"{value!r} {number_format}", value])
        if number_format is not None:
            ws.cell(row=i + 2, column=4).number_format = number_format
    wb.save(path)


def read(path):
    return [row.get("default") for row in xlsx_to_dict(path)["survey"]]


if __name__ == "__main__":
    if sys.argv[1:] == ["write"]:
        path = os.path.join(HERE, "cells.xlsx")
        build(path)
        with open(os.path.join(HERE, "cells.json"), "w") as f:
            json.dump(read(path), f, indent=1)
            f.write("\n")
    elif len(sys.argv) == 3 and sys.argv[1] == "read":
        json.dump(read(sys.argv[2]), sys.stdout)
    else:
        sys.exit(__doc__)
