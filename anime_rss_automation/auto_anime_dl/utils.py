import json
import logging
import os
from typing import Any, Dict, Optional


def ensure_dir(path: str) -> None:
    """Create directory if not exists."""
    if not path:
        return
    os.makedirs(path, exist_ok=True)


def read_json(file_path: str) -> Dict[str, Any]:
    """Read JSON file and return dict; return {} on failure."""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as exc:
        logging.error("Failed to read JSON %s: %s", file_path, exc)
        return {}


def write_json(data: Any, file_path: str) -> bool:
    """Write data as JSON to file; return True on success."""
    try:
        ensure_dir(os.path.dirname(os.path.abspath(file_path)))
        with open(file_path, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
        return True
    except Exception as exc:
        logging.error("Failed to write JSON %s: %s", file_path, exc)
        return False


def get_env(name: str, default: Optional[str] = None) -> Optional[str]:
    """Get environment variable with default."""
    value = os.getenv(name)
    return value if value is not None else default


