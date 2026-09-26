"""Checking a card against the specification, without refusing to read it.

The reading pipeline puts validation after version detection and before normalisation, which
means a malformed card still produces a :class:`~ccv3.model.Card` and the problems come back
as a list of issues rather than as an exception.

That is a deliberate position, and it is the one the ecosystem has settled on. Every real
implementation in the field is permissive to a degree that would surprise someone reading the
specification for the first time: the dominant application's own V3 validator checks only that
``data`` is an object. The reason is that a card with a malformed field is still usually a
usable card, and a reader that refuses it produces a worse outcome for the user than one that
imports it and says what looked wrong.

So this module exists to *report*, and callers who want the strict behaviour can turn issues
of severity ``error`` into exceptions themselves.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional

from .version import (
    SPEC_LOREBOOK_V3,
    SPEC_V1,
    SPEC_V2,
    SPEC_V3,
    is_newer_major,
    parse_version,
)

#: Issue severities. ``error`` means the specification is definitely violated. ``warning``
#: means something is missing, oddly typed, or unusual.
SEVERITY_ERROR = "error"
SEVERITY_WARNING = "warning"

#: The fields V3 requires inside ``data``.
V3_REQUIRED_DATA_FIELDS = (
    "name",
    "description",
    "personality",
    "scenario",
    "first_mes",
    "mes_example",
    "creator_notes",
    "system_prompt",
    "post_history_instructions",
    "alternate_greetings",
    "tags",
    "creator",
    "character_version",
    "extensions",
    "group_only_greetings",
)

#: The fields V2 requires inside ``data``. The same list without the V3 addition.
V2_REQUIRED_DATA_FIELDS = V3_REQUIRED_DATA_FIELDS[:-1]

#: The fields a V1 card must carry at the top level.
V1_REQUIRED_FIELDS = ("name", "description", "personality", "scenario", "first_mes", "mes_example")

#: The fields a lorebook entry must carry.
ENTRY_REQUIRED_FIELDS = ("keys", "content", "extensions", "enabled", "insertion_order")

#: Fields that must be arrays of strings.
STRING_ARRAY_FIELDS = ("alternate_greetings", "tags", "group_only_greetings", "source")

#: The uri schemes the specification defines for an asset.
KNOWN_ASSET_URI_SCHEMES = ("embeded://", "ccdefault:", "https://", "http://", "data:")


@dataclass
class Issue:
    """One problem found in a card."""

    severity: str
    path: str
    message: str

    def __str__(self) -> str:
        return f"[{self.severity}] {self.path}: {self.message}"


@dataclass
class ValidationReport:
    """Every issue found, with convenience accessors."""

    issues: list[Issue] = field(default_factory=list)

    @property
    def errors(self) -> list[Issue]:
        return [i for i in self.issues if i.severity == SEVERITY_ERROR]

    @property
    def warnings(self) -> list[Issue]:
        return [i for i in self.issues if i.severity == SEVERITY_WARNING]

    @property
    def ok(self) -> bool:
        return not self.errors

    def messages(self) -> list[str]:
        """Human-readable strings, for attaching to a card's warning list."""
        return [str(issue) for issue in self.issues]

    def add(self, severity: str, path: str, message: str) -> None:
        self.issues.append(Issue(severity=severity, path=path, message=message))


def _check_string_array(report: ValidationReport, container: dict[str, Any], key: str, path: str) -> None:
    if key not in container:
        return
    value = container[key]
    if not isinstance(value, list):
        report.add(
            SEVERITY_ERROR,
            f"{path}.{key}",
            f"must be an array, found {type(value).__name__}",
        )
        return
    for index, item in enumerate(value):
        if not isinstance(item, str):
            report.add(
                SEVERITY_WARNING,
                f"{path}.{key}[{index}]",
                f"expected a string, found {type(item).__name__}",
            )


def validate_lorebook(report: ValidationReport, book: Any, path: str = "data.character_book") -> None:
    """Check a lorebook and its entries."""
    if not isinstance(book, dict):
        report.add(SEVERITY_ERROR, path, f"must be an object, found {type(book).__name__}")
        return

    if "entries" not in book:
        report.add(SEVERITY_ERROR, path, "is missing the required 'entries' array")
        return
    if not isinstance(book["entries"], list):
        report.add(SEVERITY_ERROR, f"{path}.entries", "must be an array")
        return

    for key in ("extensions",):
        if key not in book:
            report.add(SEVERITY_WARNING, path, f"is missing the required '{key}' object")

    for index, entry in enumerate(book["entries"]):
        entry_path = f"{path}.entries[{index}]"
        if not isinstance(entry, dict):
            report.add(
                SEVERITY_ERROR, entry_path, f"must be an object, found {type(entry).__name__}"
            )
            continue

        for key in ENTRY_REQUIRED_FIELDS:
            if key not in entry:
                severity = SEVERITY_WARNING if key in ("extensions",) else SEVERITY_ERROR
                report.add(severity, entry_path, f"is missing the required '{key}' field")

        _check_string_array(report, entry, "keys", entry_path)
        _check_string_array(report, entry, "secondary_keys", entry_path)

        position = entry.get("position")
        if position is not None and position not in ("before_char", "after_char"):
            report.add(
                SEVERITY_ERROR,
                f"{entry_path}.position",
                f"must be 'before_char' or 'after_char', found {position!r}",
            )

        if "use_regex" in entry and not isinstance(entry["use_regex"], bool):
            report.add(SEVERITY_WARNING, f"{entry_path}.use_regex", "should be a boolean")

        if "enabled" in entry and not isinstance(entry["enabled"], bool):
            report.add(SEVERITY_WARNING, f"{entry_path}.enabled", "should be a boolean")


def validate_assets(report: ValidationReport, assets: Any, path: str = "data.assets") -> None:
    """Check the asset array, including the URI forms and the main-icon rule."""
    if assets is None:
        return
    if not isinstance(assets, list):
        report.add(SEVERITY_ERROR, path, f"must be an array, found {type(assets).__name__}")
        return

    main_icons = 0
    for index, asset in enumerate(assets):
        asset_path = f"{path}[{index}]"
        if not isinstance(asset, dict):
            report.add(SEVERITY_ERROR, asset_path, "must be an object")
            continue

        for key in ("type", "uri", "name", "ext"):
            if key not in asset:
                report.add(SEVERITY_WARNING, asset_path, f"is missing the '{key}' field")

        asset_type = asset.get("type")
        if isinstance(asset_type, str):
            known = ("icon", "background", "emotion", "user_icon", "other")
            if asset_type not in known and not asset_type.startswith("x_"):
                report.add(
                    SEVERITY_WARNING,
                    f"{asset_path}.type",
                    f"{asset_type!r} is not a standard type and does not begin with 'x_'",
                )
            if asset_type == "icon" and asset.get("name") == "main":
                main_icons += 1

        uri = asset.get("uri")
        if isinstance(uri, str) and not uri.startswith(KNOWN_ASSET_URI_SCHEMES):
            report.add(
                SEVERITY_WARNING,
                f"{asset_path}.uri",
                f"{uri!r} does not begin with one of the schemes the specification defines "
                f"({', '.join(KNOWN_ASSET_URI_SCHEMES)})",
            )

    if main_icons > 1:
        report.add(
            SEVERITY_ERROR,
            path,
            f"has {main_icons} 'icon' assets named 'main', but the name must be unique",
        )


def validate_card_dict(
    obj: Any,
    *,
    spec: Optional[str] = None,
    version: Optional[float] = None,
) -> ValidationReport:
    """Validate a decoded card object against its specification.

    When ``spec`` and ``version`` are omitted they are taken from the object itself, so this
    function is usable directly on parsed JSON without running the rest of the pipeline.
    """
    report = ValidationReport()

    if not isinstance(obj, dict):
        report.add(SEVERITY_ERROR, "", f"a card must be a JSON object, found {type(obj).__name__}")
        return report

    if spec is None:
        from .version import detect_version

        try:
            spec, version, _warnings = detect_version(obj)
        except Exception as exc:  # noqa: BLE001 - reported as an issue, not raised
            report.add(SEVERITY_ERROR, "", str(exc))
            return report

    if version is None:
        version = parse_version(obj.get("spec_version")) or 0.0

    if spec == SPEC_LOREBOOK_V3:
        validate_lorebook(report, obj.get("data"), path="data")
        return report

    # --- the wrapper ------------------------------------------------------------------
    if spec in (SPEC_V2, SPEC_V3):
        if obj.get("spec") != spec:
            report.add(
                SEVERITY_WARNING,
                "spec",
                f"expected {spec!r} for a card of this version, found {obj.get('spec')!r}",
            )
        declared = parse_version(obj.get("spec_version"))
        if declared is None:
            report.add(SEVERITY_ERROR, "spec_version", "is missing or is not a number")
        elif declared != version:
            report.add(
                SEVERITY_WARNING,
                "spec_version",
                f"declares {declared:g} but was read as {version:g}",
            )
        if is_newer_major(version):
            report.add(
                SEVERITY_WARNING,
                "spec_version",
                f"declares a major version ahead of {SPEC_V3}; unknown fields will be "
                f"preserved but not acted on",
            )

    # --- the data object --------------------------------------------------------------
    if spec == SPEC_V1:
        for key in V1_REQUIRED_FIELDS:
            if key not in obj:
                report.add(SEVERITY_WARNING, "", f"is missing the '{key}' field expected of a V1 card")
        return report

    if "data" not in obj:
        report.add(SEVERITY_ERROR, "", "is missing the required 'data' object")
        return report

    data = obj["data"]
    if not isinstance(data, dict):
        report.add(SEVERITY_ERROR, "data", f"must be an object, found {type(data).__name__}")
        return report

    required = V3_REQUIRED_DATA_FIELDS if spec == SPEC_V3 else V2_REQUIRED_DATA_FIELDS
    for key in required:
        if key not in data:
            # A card written against an earlier version legitimately lacks the V3 additions,
            # so those are a warning and the shared fields are an error.
            severity = (
                SEVERITY_WARNING
                if spec == SPEC_V3 and key in set(V3_REQUIRED_DATA_FIELDS) - set(V2_REQUIRED_DATA_FIELDS)
                else SEVERITY_ERROR
            )
            report.add(severity, "data", f"is missing the required '{key}' field")

    for key in STRING_ARRAY_FIELDS:
        _check_string_array(report, data, key, "data")

    for key in ("extensions",):
        if key in data and not isinstance(data[key], dict):
            report.add(SEVERITY_ERROR, f"data.{key}", "must be an object")

    if "name" in data and isinstance(data["name"], str) and not data["name"].strip():
        report.add(SEVERITY_WARNING, "data.name", "is empty, which will make the card hard to identify")

    for key in ("creation_date", "modification_date"):
        if key in data:
            value = data[key]
            if isinstance(value, bool) or not isinstance(value, (int, float)):
                report.add(SEVERITY_WARNING, f"data.{key}", "should be a number of unix seconds")

    multilingual = data.get("creator_notes_multilingual")
    if multilingual is not None:
        if not isinstance(multilingual, dict):
            report.add(SEVERITY_ERROR, "data.creator_notes_multilingual", "must be an object")
        else:
            for key in multilingual:
                if len(key) != 2 or not key.isalpha():
                    report.add(
                        SEVERITY_WARNING,
                        f"data.creator_notes_multilingual.{key}",
                        "is not a two-letter ISO 639-1 language code",
                    )

    if spec == SPEC_V3:
        validate_assets(report, data.get("assets"))

    if "character_book" in data:
        validate_lorebook(report, data["character_book"])

    return report
