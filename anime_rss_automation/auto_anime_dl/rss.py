import os
import re
import ssl
import time
import requests
import feedparser
from bs4 import BeautifulSoup
from collections import defaultdict
from typing import Dict, Any, Optional


def create_ssl_context():
    context = ssl.create_default_context()
    context.check_hostname = False
    context.verify_mode = ssl.CERT_NONE
    ssl._create_default_https_context = lambda: context  # type: ignore[attr-defined]
    return context


def get_rss_feed(url: str):
    try:
        return feedparser.parse(url)
    except Exception:
        return None


def extract_anime_info(title: str) -> Dict[str, Optional[str]]:
    episode_match = re.search(r"-\s*(\d+)\s*\[", title)
    episode = f"E{episode_match.group(1)}" if episode_match else None
    name_match = re.search(r"(?<=]\s)(.*?)\s*(?=\s*-)", title)
    anime_name = name_match.group(1).strip() if name_match else title
    anime_name = re.sub(r"\[.*?\]", "", anime_name).strip()
    return {"title": anime_name, "episode": episode}


def extract_rss_data(feed_data) -> Dict[str, Any]:
    if not feed_data or feed_data.bozo or not feed_data.get("entries"):
        return {}
    bangumi = defaultdict(list)
    for entry in feed_data.entries:
        title = entry.get("title", "无标题")
        anime_info = extract_anime_info(title)
        enclosures = entry.get("enclosures", [])
        media_url = enclosures[0].href if enclosures else None
        if not media_url:
            continue
        page_link = entry.get("link", "")
        episode_info = {
            "filename": f"{anime_info['title']} {anime_info['episode'] or ''}.mp4".strip(),
            "bt_url": media_url,
            "pubDate": entry.get("published", "无发布日期"),
            "original_title": title,
            "page_link": page_link,
        }
        anime_title = anime_info["title"]
        if not anime_info["episode"]:
            match = re.search(r"[Ee]([0-9]+)", media_url)
            if match:
                episode_info["filename"] = f"{anime_title} E{match.group(1)}.mp4"
        bangumi[anime_title].append(episode_info)
    return dict(bangumi)


def fetch_cover_from_link(page_url: str) -> Optional[str]:
    try:
        response = requests.get(page_url, timeout=10)
        response.raise_for_status()
        soup = BeautifulSoup(response.text, "html.parser")
        description_div = soup.find("div", class_="description")
        if not description_div:
            return None
        img_tag = description_div.find("img")
        if not img_tag:
            return None
        cover_url = img_tag.get("src")
        return cover_url or None
    except Exception:
        return None


def enhance_bangumi_data(raw_data: Dict[str, Any]) -> Dict[str, Any]:
    enhanced_data: Dict[str, Any] = {}
    for anime_title, episodes in raw_data.get("Bangumi", {}).items():
        if not episodes:
            continue
        page_link = episodes[0].get("page_link", "")
        cover_url = fetch_cover_from_link(page_link) if page_link else None
        enhanced_data[anime_title] = {
            "episodes": episodes,
            "cover_info": {"cover_url": cover_url or ""} if cover_url else {},
        }
        time.sleep(1)
    return enhanced_data


