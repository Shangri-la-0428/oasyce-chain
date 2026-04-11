#!/usr/bin/env python3
"""SDK-native compatibility helpers for chain-side demo agents.

These scripts stay in ``oasyce-chain`` only as thin wrappers for demos, E2E,
and backward compatibility. The canonical AI runtime lives in ``oasyce-sdk``.
"""

from __future__ import annotations

from dataclasses import dataclass
from importlib import import_module
from importlib.metadata import PackageNotFoundError, version as dist_version
import os
from pathlib import Path
import re
import sys
from typing import Any


SDK_TESTED_BASELINE = "0.12.0"
_SDK_MODES = {"source", "installed", "auto"}
_REQUIRED_SIGNER_METHODS = (
    "register_capability",
    "invoke_capability",
    "complete_invocation",
    "claim_invocation",
    "register_asset",
    "buy_shares",
    "submit_feedback",
    "set_delegate_policy",
    "enroll_delegate",
    "create_sigil",
    "pulse_sigil",
)
_PULSE_TYPE_URL = "/oasyce.sigil.v1.MsgPulse"

_SDK_INSTALL_HINT = (
    "oasyce-sdk is required for SDK-native agent writes. "
    "Install it with `pip install -U \"oasyce-sdk>=0.12.0\"`, "
    "or point OASYCE_SDK_PATH at a local checkout."
)

_IDENTITY_HINT = (
    "No local SDK identity is ready for chain writes. "
    "Run `oasyce start` on the first device, `oasyce join` on a receiving device, "
    "or set OASYCE_MNEMONIC for a headless runtime."
)


def _sdk_candidate_paths() -> list[Path]:
    paths: list[Path] = []
    env_path = os.environ.get("OASYCE_SDK_PATH", "").strip()
    if env_path:
        paths.append(Path(env_path).expanduser())
    repo_root = Path(__file__).resolve().parent.parent
    paths.append(repo_root.parent / "oasyce-sdk")
    return paths


def _normalize_sdk_mode(mode: str | None = None) -> str:
    value = (mode or os.environ.get("OASYCE_SDK_MODE", "source")).strip().lower()
    if value not in _SDK_MODES:
        raise RuntimeError(
            f"Unsupported OASYCE_SDK_MODE {value!r}; expected one of: {', '.join(sorted(_SDK_MODES))}"
        )
    return value


def _source_checkout_path() -> Path | None:
    for candidate in _sdk_candidate_paths():
        if (candidate / "oasyce_sdk").exists():
            return candidate
    return None


def _read_pyproject_version(path: Path | None) -> str | None:
    if path is None:
        return None
    pyproject = path / "pyproject.toml"
    if not pyproject.exists():
        return None
    match = re.search(
        r'^version\s*=\s*"([^"]+)"',
        pyproject.read_text(encoding="utf-8"),
        re.MULTILINE,
    )
    return match.group(1) if match else None


def _safe_dist_version() -> str | None:
    try:
        return dist_version("oasyce-sdk")
    except PackageNotFoundError:
        return None


def _version_tuple(value: str | None) -> tuple[int, ...] | None:
    if not value:
        return None
    parts = re.findall(r"\d+", value)
    return tuple(int(part) for part in parts) if parts else None


def _version_lt(value: str | None, baseline: str) -> bool:
    parsed = _version_tuple(value)
    target = _version_tuple(baseline)
    if parsed is None or target is None:
        return False
    width = max(len(parsed), len(target))
    parsed = parsed + (0,) * (width - len(parsed))
    target = target + (0,) * (width - len(target))
    return parsed < target


def _is_relative_to(path: Path, base: Path) -> bool:
    try:
        path.resolve().relative_to(base.resolve())
        return True
    except ValueError:
        return False


def _ensure_sdk_importable(mode: str | None = None) -> None:
    requested_mode = _normalize_sdk_mode(mode)
    source_checkout = _source_checkout_path()

    if requested_mode in {"source", "auto"} and source_checkout is not None:
        source_str = str(source_checkout)
        if sys.path[:1] != [source_str]:
            sys.path.insert(0, source_str)

    try:
        import oasyce_sdk  # noqa: F401
        return
    except ImportError:
        pass

    if requested_mode == "installed":
        raise RuntimeError(_SDK_INSTALL_HINT)

    for candidate in _sdk_candidate_paths():
        if (candidate / "oasyce_sdk").exists():
            sys.path.insert(0, str(candidate))
            break

    try:
        import oasyce_sdk  # noqa: F401
    except ImportError as exc:
        raise RuntimeError(_SDK_INSTALL_HINT) from exc


def _load_sdk_contract(mode: str | None = None) -> dict[str, Any]:
    requested_mode = _normalize_sdk_mode(mode)
    _ensure_sdk_importable(requested_mode)

    sdk_module = import_module("oasyce_sdk")
    from oasyce_sdk import OasyceClient
    from oasyce_sdk.crypto import NativeSigner

    contract: dict[str, Any] = {
        "sdk_module": sdk_module,
        "OasyceClient": OasyceClient,
        "NativeSigner": NativeSigner,
        "seam": "public",
    }

    try:
        from oasyce_sdk.sigil import Identity, resolve_identity

        contract["Identity"] = Identity
        contract["resolve_identity"] = resolve_identity
        return contract
    except Exception as exc:  # noqa: BLE001
        contract["public_seam_error"] = str(exc)

    from oasyce_sdk.delegate_policy import ensure_chain_identity
    from oasyce_sdk.identity import IdentityResolver

    def _compat_resolve_identity(*, client, chain_id):
        identity = IdentityResolver.resolve()
        return ensure_chain_identity(identity, client, chain_id)

    contract["seam"] = "compat_bridge"
    contract["Identity"] = None
    contract["resolve_identity"] = _compat_resolve_identity
    contract["compat_bridge"] = {
        "ensure_chain_identity": ensure_chain_identity,
        "IdentityResolver": IdentityResolver,
    }
    return contract


def inspect_sdk_surface(mode: str | None = None) -> dict[str, Any]:
    requested_mode = _normalize_sdk_mode(mode)
    contract = _load_sdk_contract(requested_mode)
    sdk_module = contract["sdk_module"]
    module_path = Path(getattr(sdk_module, "__file__", "")).resolve()
    source_checkout = _source_checkout_path()
    source_version = _read_pyproject_version(source_checkout)
    package_version = getattr(sdk_module, "__version__", None)
    installed_dist_version = _safe_dist_version()
    resolved_from_source = source_checkout is not None and _is_relative_to(module_path, source_checkout)

    try:
        scanner_module = import_module("oasyce_sdk.agent.scanner")
        scanner_scan = callable(getattr(scanner_module, "scan", None))
    except Exception as exc:  # noqa: BLE001
        scanner_module = None
        scanner_scan = False
        scanner_error = str(exc)
    else:
        scanner_error = ""

    signer_cls = contract["NativeSigner"]
    signer_methods = {name: hasattr(signer_cls, name) for name in _REQUIRED_SIGNER_METHODS}

    try:
        from oasyce_sdk.crypto.msg_schemas import MSG_SCHEMAS
    except Exception as exc:  # noqa: BLE001
        pulse_schema_present = False
        pulse_schema_has_dimensions = False
        pulse_schema_fields = []
        pulse_schema_error = str(exc)
    else:
        pulse_schema_fields = MSG_SCHEMAS.get(_PULSE_TYPE_URL, [])
        pulse_schema_present = _PULSE_TYPE_URL in MSG_SCHEMAS
        pulse_schema_has_dimensions = any(field[0] == "dimensions" for field in pulse_schema_fields)
        pulse_schema_error = ""

    pulse_helpers = sorted(name for name in dir(signer_cls) if "pulse" in name.lower())

    warnings: list[str] = []
    errors: list[str] = []

    if requested_mode == "source":
        if source_checkout is None:
            errors.append("source mode requested but no adjacent oasyce-sdk checkout was found")
        elif not resolved_from_source:
            errors.append(
                "source mode requested but imported oasyce_sdk did not resolve from the adjacent checkout"
            )

    if resolved_from_source and source_version and package_version and package_version != source_version:
        errors.append(
            f"editable metadata drift: package __version__={package_version}, source pyproject={source_version}"
        )
    if resolved_from_source and source_version and installed_dist_version and installed_dist_version != source_version:
        warnings.append(
            f"distribution metadata drift: installed dist version={installed_dist_version}, source pyproject={source_version}"
        )
    if source_version and _version_lt(source_version, SDK_TESTED_BASELINE):
        errors.append(
            f"source checkout version {source_version} is below the chain-side tested baseline {SDK_TESTED_BASELINE}"
        )
    if not resolved_from_source and installed_dist_version and _version_lt(installed_dist_version, SDK_TESTED_BASELINE):
        warnings.append(
            f"installed SDK version {installed_dist_version} is below the chain-side tested baseline {SDK_TESTED_BASELINE}"
        )

    if contract["seam"] != "public":
        errors.append("public identity seam unavailable; using compatibility bridge")
    if not scanner_scan:
        errors.append("oasyce_sdk.agent.scanner.scan is unavailable")
        if scanner_error:
            warnings.append(f"scanner import error: {scanner_error}")

    missing_signer = sorted(name for name, present in signer_methods.items() if not present)
    if missing_signer:
        errors.append(f"NativeSigner is missing required methods: {', '.join(missing_signer)}")

    if "pulse_sigil" not in pulse_helpers:
        errors.append("SDK has no explicit pulse_sigil helper on NativeSigner")
    if pulse_schema_present and not pulse_schema_has_dimensions:
        errors.append("SDK MsgPulse schema is present but does not encode the dimensions map field")
    if not pulse_schema_present:
        errors.append("SDK MsgPulse schema is missing from msg_schemas.py")
    if pulse_schema_error:
        errors.append(f"Could not inspect SDK pulse schema: {pulse_schema_error}")

    status = "error" if errors else "warn" if warnings else "ok"
    return {
        "status": status,
        "requested_mode": requested_mode,
        "tested_baseline": SDK_TESTED_BASELINE,
        "source_checkout": str(source_checkout) if source_checkout is not None else "",
        "source_checkout_version": source_version or "",
        "resolved_from_source": resolved_from_source,
        "module_path": str(module_path),
        "package_version": package_version or "",
        "distribution_version": installed_dist_version or "",
        "identity_seam": contract["seam"],
        "public_identity_seam": contract["seam"] == "public",
        "public_seam_error": contract.get("public_seam_error", ""),
        "scanner_module": getattr(scanner_module, "__file__", "") if scanner_module is not None else "",
        "scanner_scan": scanner_scan,
        "signer_methods": signer_methods,
        "pulse": {
            "helper_names": pulse_helpers,
            "schema_present": pulse_schema_present,
            "schema_has_dimensions": pulse_schema_has_dimensions,
            "schema_field_count": len(pulse_schema_fields),
            "schema_error": pulse_schema_error,
        },
        "warnings": warnings,
        "errors": errors,
    }


@dataclass
class SDKRuntime:
    chain_rest: str
    identity: Any
    chain_id: str
    client_cls: Any
    signer_cls: Any
    identity_seam: str = "public"

    @property
    def principal(self) -> str:
        context = getattr(self.identity, "context", None)
        return getattr(context, "principal", None) or getattr(self.identity, "principal", "") or ""

    @property
    def wallet(self):
        context = getattr(self.identity, "context", None)
        return getattr(context, "wallet", None) or getattr(self.identity, "wallet", None)

    @property
    def actor_address(self) -> str:
        return self.principal or self.signer_address

    @property
    def signer_address(self) -> str:
        return getattr(self.identity, "address", "") or getattr(getattr(self.identity, "context", None), "address", "")

    @property
    def is_present(self) -> bool:
        if hasattr(self.identity, "is_present"):
            return bool(self.identity.is_present)
        return bool(self.wallet)

    @property
    def is_delegated(self) -> bool:
        return bool(self.actor_address and self.signer_address and self.actor_address != self.signer_address)

    @property
    def client(self):
        """Return a fresh client to avoid cross-thread Session reuse."""
        return self.client_cls(self.chain_rest)

    @property
    def signer(self):
        """Return a fresh signer for each write to avoid shared mutable caches."""
        if self.wallet is not None:
            return self.signer_cls(
                self.wallet,
                self.client,
                chain_id=self.chain_id,
                principal=self.principal or None,
            )
        signer = getattr(self.identity, "signer", None)
        if signer is not None:
            return signer
        raise RuntimeError(_IDENTITY_HINT)


def resolve_runtime(chain_rest: str, chain_id: str, mode: str | None = None) -> SDKRuntime:
    """Resolve the canonical SDK-native write path for this device."""
    contract = _load_sdk_contract(mode)
    client_cls = contract["OasyceClient"]
    signer_cls = contract["NativeSigner"]

    try:
        client = client_cls(chain_rest)
        identity = contract["resolve_identity"](client=client, chain_id=chain_id)
        if hasattr(identity, "is_present") and not identity.is_present:
            raise RuntimeError(_IDENTITY_HINT)
        if not getattr(identity, "address", ""):
            raise RuntimeError(_IDENTITY_HINT)
        return SDKRuntime(
            chain_rest=chain_rest,
            identity=identity,
            chain_id=chain_id,
            client_cls=client_cls,
            signer_cls=signer_cls,
            identity_seam=contract["seam"],
        )
    except FileNotFoundError as exc:
        raise RuntimeError(_IDENTITY_HINT) from exc
    except RuntimeError as exc:
        message = str(exc)
        if "No identity found" in message:
            raise RuntimeError(_IDENTITY_HINT) from exc
        raise


def runtime_sdk_report(runtime: SDKRuntime, mode: str | None = None) -> dict[str, Any]:
    surface = inspect_sdk_surface(mode)
    surface["runtime"] = {
        "identity_seam": runtime.identity_seam,
        "actor_address": runtime.actor_address,
        "signer_address": runtime.signer_address,
        "delegated": runtime.is_delegated,
    }
    return surface


def submit_single(
    runtime: SDKRuntime,
    type_url: str,
    fields: dict[str, Any],
    *,
    delegate_exec: bool = True,
):
    """Broadcast one message, mirroring the SDK's canonical delegate wrapping."""
    signer = runtime.signer
    if delegate_exec and runtime.is_delegated:
        return signer.sign_and_broadcast([(
            "/oasyce.delegate.v1.MsgExec",
            {
                "delegate": runtime.signer_address,
                "msgs": [{"@type": type_url, **fields}],
            },
        )])
    return signer.sign_and_broadcast([(type_url, fields)])


def tx_status(result) -> tuple[bool, str]:
    """Normalize an SDK TxResult into the legacy (ok, detail) shape."""
    if result.success:
        return True, result.tx_hash or "submitted"
    detail = result.raw_log or f"code={result.code}"
    return False, detail


def split_csv(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]


def parse_uoas(value: Any) -> int:
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    if isinstance(value, str):
        text = value.strip().lower()
        if text.isdigit():
            return int(text)
        if text.endswith("uoas"):
            amount = text[:-4].strip()
            return int(amount) if amount.isdigit() else 0
        match = re.match(r"^([0-9]+(?:\.[0-9]+)?)\s*oas$", text)
        if match:
            return int(float(match.group(1)) * 1_000_000)
    return 0
