"""Worker configuration, read lazily from the environment.

Nothing here is read at import time: importing the module must never fail,
because the process has to be able to log which variable is missing. The
required variables are validated once, when :func:`load_config` (or
:meth:`WorkerConfig.from_env`) is called during startup.
"""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass

DEFAULT_HOURLY_RATE_CENTS = 8000
DEFAULT_VAT_PERCENT = 19
DEFAULT_INVOICE_QUEUE_KEY = "invoices"


class ConfigError(RuntimeError):
    """Raised when a worker variable is missing or malformed."""


def _required(env: Mapping[str, str], name: str) -> str:
    value = env.get(name)
    if value is None or value == "":
        raise ConfigError(
            f"{name} is not set; declare it in RUN.json and export it before starting the worker"
        )
    return value


def _int_env(env: Mapping[str, str], name: str, default: int) -> int:
    raw = env.get(name)
    if raw is None or raw == "":
        return default
    try:
        return int(raw)
    except ValueError as exc:
        raise ConfigError(f"{name} must be an integer, got {raw!r}") from exc


@dataclass(frozen=True)
class WorkerConfig:
    """Every worker variable of the sprint contract."""

    database_url: str
    valkey_url: str
    hourly_rate_cents: int
    vat_percent: int
    invoice_queue_key: str

    @classmethod
    def from_env(cls, env: Mapping[str, str] | None = None) -> WorkerConfig:
        source = os.environ if env is None else env
        return cls(
            database_url=_required(source, "DATABASE_URL"),
            valkey_url=_required(source, "VALKEY_URL"),
            hourly_rate_cents=_int_env(source, "HOURLY_RATE_CENTS", DEFAULT_HOURLY_RATE_CENTS),
            vat_percent=_int_env(source, "VAT_PERCENT", DEFAULT_VAT_PERCENT),
            invoice_queue_key=source.get("INVOICE_QUEUE_KEY") or DEFAULT_INVOICE_QUEUE_KEY,
        )


def load_config() -> WorkerConfig:
    """Read and validate the worker configuration from the process environment."""
    return WorkerConfig.from_env()
