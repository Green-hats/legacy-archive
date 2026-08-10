import os
from pathlib import Path


BASE_DIR = Path(__file__).resolve().parent.parent

PIKPAK_USERNAME = os.getenv("PIKPAK_USERNAME", "")
PIKPAK_PASSWORD = os.getenv("PIKPAK_PASSWORD", "")
ROOT_FOLDER_ID = os.getenv("PIKPAK_ROOT_FOLDER_ID", "")

RSS_URLS = {
    "anime_collection": os.getenv(
        "RSS_URL_ANIME_COLLECTION",
        "https://api.animes.garden/collection/your-id/feed.xml",
    )
}

BANGUMI_JSON_PATH = str(BASE_DIR / "bangumi.json")
BANGUMI_WITH_COVER_JSON_PATH = str(BASE_DIR / "bangumi_with_cover.json")
BANGUMI_WITH_PLAY_URL_PATH = str(BASE_DIR / "frontend" / "bangumi_with_play_url.json")
FOLDER_STRUCTURE_PATH = str(BASE_DIR / "pikpak_folder_structure.json")
STATE_PATH = str(BASE_DIR / ".processed_state.json")

SCHEDULE_INTERVAL_SECONDS = int(os.getenv("SCHEDULE_INTERVAL_SECONDS", "21600"))
