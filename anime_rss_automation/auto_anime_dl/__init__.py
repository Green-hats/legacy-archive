"""Auto Anime DL core package.

This package provides modularized components for:
- PikPak automation and scanning
- RSS collection and data enhancement
- JSON IO helpers and shared utilities

The package is intended to be used by thin CLI wrappers in the
repository root (e.g., `rssworker.py`, `pikpak_dl.py`, `pikpak_automation.py`).
"""

__all__ = [
    "pikpak",
    "rss",
    "utils",
]


