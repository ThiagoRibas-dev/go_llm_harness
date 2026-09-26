"""Exception types for the ccv3 reader.

Every exception carries enough context to tell the caller which stage failed, because the
three failure modes that matter are distinguishable and callers usually want to treat them
differently: the container could not be opened, the card inside it is not a card, or the card
is a card but of a version this reader does not understand.
"""

from __future__ import annotations


class CCV3Error(Exception):
    """Base class for every error this package raises."""


class ContainerError(CCV3Error):
    """The bytes handed to the reader are not a recognised container.

    Raised when the input is not a PNG, not a JSON object, and not a CHARX archive that
    survives unzipping.
    """


class MalformedCardError(CCV3Error):
    """A container was opened successfully but the payload inside it is not usable.

    The distinction from :class:`ContainerError` matters when a file contains several
    candidate payloads. A PNG with a corrupt ``ccv3`` chunk but a readable ``chara`` chunk is
    not a failure, it is a fallback with a warning, so this exception is caught internally in
    that situation and only propagates when every candidate is unusable.
    """


class UnknownSpecError(MalformedCardError):
    """The payload parsed as JSON but could not be identified as a card.

    This is the error raised when a JSON file is some other kind of JSON entirely, which
    happens often enough in a directory of downloaded files that the message should be
    specific about what was found.
    """


class UnsupportedVersionError(MalformedCardError):
    """The card identified its specification version but this reader cannot handle it.

    In practice this is only raised for a version below 1, since a *newer* major version is
    expected to be imported on a best-effort basis with a warning rather than refused. See
    the specification's guidance on comparing ``spec_version`` as a float.
    """
