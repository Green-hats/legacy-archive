import asyncio
import logging
import time
from typing import Dict, Any

from auto_anime_dl.config import (
    PIKPAK_USERNAME,
    PIKPAK_PASSWORD,
    ROOT_FOLDER_ID,
    BANGUMI_WITH_COVER_JSON_PATH,
    FOLDER_STRUCTURE_PATH,
    BANGUMI_WITH_PLAY_URL_PATH,
)
from auto_anime_dl.pikpak import PikPakScanner
from auto_anime_dl.utils import read_json, write_json


logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')


async def main() -> None:
    if not PIKPAK_USERNAME or not PIKPAK_PASSWORD or not ROOT_FOLDER_ID:
        logging.error("缺少 PikPak 配置，请设置环境变量 PIKPAK_USERNAME/PIKPAK_PASSWORD/PIKPAK_ROOT_FOLDER_ID")
        raise SystemExit(1)

    scanner = PikPakScanner(PIKPAK_USERNAME, PIKPAK_PASSWORD)
    if not await scanner.login():
        logging.error("无法登录PikPak，程序退出")
        return

    folder_structure = await scanner.scan_folder_structure(ROOT_FOLDER_ID)
    write_json(folder_structure, FOLDER_STRUCTURE_PATH)

    bangumi_data: Dict[str, Any] = read_json(BANGUMI_WITH_COVER_JSON_PATH)
    if not bangumi_data:
        logging.error("无法加载Bangumi数据，程序退出")
        return

    logging.info(f"成功加载 {len(bangumi_data.get('Bangumi', {}))} 部动画数据")
    new_data: Dict[str, Any] = {"Bangumi": {}, "timestamp": time.strftime("%Y-%m-%d %H:%M:%S")}

    for anime_title, anime_data in bangumi_data.get("Bangumi", {}).items():
        logging.info(f"处理动画: {anime_title}")
        new_data["Bangumi"][anime_title] = {"cover_info": anime_data.get("cover_info", {}), "episodes": []}
        episodes = anime_data.get("episodes", [])
        for episode in episodes:
            episode_num = scanner.extract_episode(episode.get("filename", ""))
            if not episode_num:
                new_data["Bangumi"][anime_title]["episodes"].append({**episode, "play_url": "", "match_status": "failed_extract_episode"})
                continue
            file_info = await scanner.find_episode_by_anime(folder_structure, anime_title, episode_num)
            play_url = file_info.get("play_url", "") if file_info else ""
            match_status = "found" if file_info and play_url else ("found_no_url" if file_info else "not_found")
            new_episode = {**episode, "play_url": play_url, "match_status": match_status}
            new_data["Bangumi"][anime_title]["episodes"].append(new_episode)
            await asyncio.sleep(0.1)

    write_json(new_data, BANGUMI_WITH_PLAY_URL_PATH)
    logging.info(f"处理完成，共处理 {len(new_data['Bangumi'])} 部动画")
    await scanner.close()


if __name__ == "__main__":
    asyncio.run(main())