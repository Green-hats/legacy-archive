import asyncio
import logging
import re
from typing import Dict, Any, Set

from .config import (
    RSS_URLS,
    STATE_PATH,
    SCHEDULE_INTERVAL_SECONDS,
    PIKPAK_USERNAME,
    PIKPAK_PASSWORD,
    ROOT_FOLDER_ID,
)
from .rss import get_rss_feed, extract_rss_data
from .state import load_processed_keys, save_processed_keys
from .pikpak import PikPakManager


def extract_infohash(magnet: str) -> str | None:
    if not magnet:
        return None
    m = re.search(r"btih:([a-fA-F0-9]{32,40})", magnet)
    return m.group(1).lower() if m else None


async def run_once() -> dict:
    if not PIKPAK_USERNAME or not PIKPAK_PASSWORD or not ROOT_FOLDER_ID:
        logging.error("缺少 PikPak 配置，跳过本轮任务")
        return {"ok": False, "reason": "missing_config"}

    processed: Set[str] = load_processed_keys(STATE_PATH)
    logging.info("已记录的历史 infohash 数量: %d", len(processed))

    # 汇总RSS
    merged: Dict[str, list[Dict[str, Any]]] = {}
    for name, url in RSS_URLS.items():
        logging.info("抓取RSS: %s (%s)", name, url)
        feed = get_rss_feed(url)
        if not feed:
            logging.warning("RSS 失败: %s", url)
            continue
        data = extract_rss_data(feed) or {}
        for title, eps in data.items():
            merged.setdefault(title, []).extend(eps)

    total_titles = len(merged)
    total_eps = sum(len(v) for v in merged.values())
    logging.info("RSS 合并后：番剧 %d，条目 %d", total_titles, total_eps)

    # 过滤出未处理的条目
    filtered: Dict[str, list[Dict[str, Any]]] = {}
    no_hash = 0
    for title, eps in merged.items():
        for ep in eps:
            bt = ep.get("bt_url", "")
            ih = extract_infohash(bt)
            # Fallback key: 文件名 + bt_url 组合，避免无法解析 infohash 时重复
            fallback_key = f"{title}||{ep.get('filename','')}||{bt}".lower()
            unique_key = ih or fallback_key
            if unique_key not in processed:
                ep["__unique_key"] = unique_key
                filtered.setdefault(title, []).append(ep)
            elif not ih:
                no_hash += 1
    logging.info("可新增的条目: %d，跳过（无hash）: %d", sum(len(v) for v in filtered.values()), no_hash)

    if not filtered:
        logging.info("没有发现新的剧集需要下载")
        # 确保状态文件存在
        save_processed_keys(STATE_PATH, processed)
        return {
            "ok": True,
            "rss": {"titles": total_titles, "episodes": total_eps},
            "new_items": 0,
            "skipped_no_hash": no_hash,
            "processed_total": len(processed),
        }

    # 触发下载
    manager = PikPakManager(PIKPAK_USERNAME, PIKPAK_PASSWORD, root_folder_id=ROOT_FOLDER_ID)
    if not await manager.login():
        logging.error("PikPak 登录失败，跳过")
        return
    if not await manager.setup_root_folder("Bangumi"):
        logging.error("根目录校验失败，跳过")
        return

    await manager.process_bangumi_data({title: {"episodes": eps} for title, eps in filtered.items()})

    # 更新状态
    new_keys = {ep.get("__unique_key") for eps in filtered.values() for ep in eps}
    processed.update(k for k in new_keys if k)
    save_processed_keys(STATE_PATH, processed)
    logging.info("新增记录 %d 条，累计 %d", len(new_keys), len(processed))
    return {
        "ok": True,
        "rss": {"titles": total_titles, "episodes": total_eps},
        "new_items": sum(len(v) for v in filtered.values()),
        "skipped_no_hash": no_hash,
        "processed_total": len(processed),
    }


async def run_scheduler() -> None:
    logging.info("RSS 调度器启动，周期: %ds", SCHEDULE_INTERVAL_SECONDS)
    while True:
        try:
            await run_once()
        except Exception as exc:
            logging.exception("调度器执行异常: %s", exc)
        await asyncio.sleep(SCHEDULE_INTERVAL_SECONDS)


