"""Tests for the worker configuration reader."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

import pytest
from worker.config import (
    DEFAULT_HOURLY_RATE_CENTS,
    DEFAULT_INVOICE_QUEUE_KEY,
    DEFAULT_VAT_PERCENT,
    ConfigError,
    WorkerConfig,
    load_config,
)

REQUIRED_ENV = {
    "DATABASE_URL": "postgresql://app:secret@localhost:5432/app",
    "VALKEY_URL": "redis://localhost:6379/0",
}


def test_reads_every_contract_variable() -> None:
    env = {
        **REQUIRED_ENV,
        "HOURLY_RATE_CENTS": "9500",
        "VAT_PERCENT": "19",
        "INVOICE_QUEUE_KEY": "invoices",
    }
    cfg = WorkerConfig.from_env(env)

    assert cfg.database_url == REQUIRED_ENV["DATABASE_URL"]
    assert cfg.valkey_url == REQUIRED_ENV["VALKEY_URL"]
    assert cfg.hourly_rate_cents == 9500
    assert cfg.vat_percent == 19
    assert cfg.invoice_queue_key == "invoices"


def test_optional_variables_fall_back_to_defaults() -> None:
    cfg = WorkerConfig.from_env(dict(REQUIRED_ENV))

    assert cfg.hourly_rate_cents == DEFAULT_HOURLY_RATE_CENTS
    assert cfg.vat_percent == DEFAULT_VAT_PERCENT
    assert cfg.invoice_queue_key == DEFAULT_INVOICE_QUEUE_KEY


@pytest.mark.parametrize("missing", ["DATABASE_URL", "VALKEY_URL"])
def test_missing_required_variable_names_itself(missing: str) -> None:
    env = {key: value for key, value in REQUIRED_ENV.items() if key != missing}

    with pytest.raises(ConfigError) as excinfo:
        WorkerConfig.from_env(env)

    assert missing in str(excinfo.value)


@pytest.mark.parametrize("name", ["HOURLY_RATE_CENTS", "VAT_PERCENT"])
def test_malformed_integer_names_itself(name: str) -> None:
    env = {**REQUIRED_ENV, name: "not-a-number"}

    with pytest.raises(ConfigError) as excinfo:
        WorkerConfig.from_env(env)

    assert name in str(excinfo.value)


def test_import_does_not_require_environment() -> None:
    """Importing the config module must not read the environment.

    A process that imports it with no worker variables set (and could not have
    read them lazily at import) has to succeed, otherwise a missing variable
    kills the worker before it can log which one.
    """
    project_dir = Path(__file__).resolve().parents[1]
    env = {
        k: v
        for k, v in os.environ.items()
        if k not in {"DATABASE_URL", "VALKEY_URL", "HOURLY_RATE_CENTS", "VAT_PERCENT"}
    }
    result = subprocess.run(
        [sys.executable, "-c", "import worker.config; import worker.main"],
        cwd=project_dir,
        env=env,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr


def test_load_config_reads_process_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@db:5432/app")
    monkeypatch.setenv("VALKEY_URL", "redis://queue:6379/0")
    monkeypatch.delenv("HOURLY_RATE_CENTS", raising=False)

    cfg = load_config()

    assert cfg.database_url == "postgresql://app@db:5432/app"
    assert cfg.valkey_url == "redis://queue:6379/0"
    assert cfg.hourly_rate_cents == DEFAULT_HOURLY_RATE_CENTS
