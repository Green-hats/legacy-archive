import json
import logging
from typing import Set


def load_processed_infohashes(state_path: str) -> Set[str]:
    try:
        with open(state_path, "r", encoding="utf-8") as f:
            data = json.load(f)
        items = set(data.get("processed_infohashes", []))
        return {h.lower() for h in items}
    except FileNotFoundError:
        return set()
    except Exception as exc:
        logging.error("Failed to load state %s: %s", state_path, exc)
        return set()


def save_processed_infohashes(state_path: str, infohashes: Set[str]) -> bool:
    try:
        with open(state_path, "w", encoding="utf-8") as f:
            json.dump({"processed_infohashes": sorted(h.lower() for h in infohashes)}, f, indent=2, ensure_ascii=False)
        return True
    except Exception as exc:
        logging.error("Failed to save state %s: %s", state_path, exc)
        return False


def load_processed_keys(state_path: str) -> Set[str]:
    """Load generic processed keys. Backward compatible with old infohash-only state."""
    try:
        with open(state_path, "r", encoding="utf-8") as f:
            data = json.load(f)
        keys = set(data.get("processed_keys", []))
        # Backward compatibility: merge old field
        keys |= set(data.get("processed_infohashes", []))
        return {k.lower() for k in keys}
    except FileNotFoundError:
        return set()
    except Exception as exc:
        logging.error("Failed to load state %s: %s", state_path, exc)
        return set()


def save_processed_keys(state_path: str, keys: Set[str]) -> bool:
    try:
        with open(state_path, "w", encoding="utf-8") as f:
            json.dump({"processed_keys": sorted(k.lower() for k in keys)}, f, indent=2, ensure_ascii=False)
        return True
    except Exception as exc:
        logging.error("Failed to save state %s: %s", state_path, exc)
        return False


