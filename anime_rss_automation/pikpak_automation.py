import asyncio
import logging
from typing import Dict, Any

from auto_anime_dl.config import PIKPAK_USERNAME, PIKPAK_PASSWORD, ROOT_FOLDER_ID, BANGUMI_WITH_COVER_JSON_PATH
from auto_anime_dl.pikpak import PikPakManager
from auto_anime_dl.utils import read_json


logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('pikpak_automation.log'),
        logging.StreamHandler()
    ]
)


def load_bangumi_data(file_path: str) -> Dict[str, Any]:
    data = read_json(file_path)
    return data.get("Bangumi", {}) if isinstance(data, dict) else {}


async def main() -> None:
    if not PIKPAK_USERNAME or not PIKPAK_PASSWORD or not ROOT_FOLDER_ID:
        logging.error("缺少 PikPak 配置，请设置环境变量 PIKPAK_USERNAME/PIKPAK_PASSWORD/PIKPAK_ROOT_FOLDER_ID")
        raise SystemExit(1)

    pikpak_manager = PikPakManager(PIKPAK_USERNAME, PIKPAK_PASSWORD, root_folder_id=ROOT_FOLDER_ID)
    if not await pikpak_manager.login():
        logging.error("无法登录PikPak，程序退出")
        raise SystemExit(1)

    if not await pikpak_manager.setup_root_folder("Bangumi"):
        logging.error("根文件夹验证失败，程序退出")
        raise SystemExit(1)

    bangumi_data = load_bangumi_data(BANGUMI_WITH_COVER_JSON_PATH)
    if not bangumi_data:
        logging.error("无法加载Bangumi数据，程序退出")
        raise SystemExit(1)

    logging.info(f"成功加载 {len(bangumi_data)} 部动画数据")
    await pikpak_manager.process_bangumi_data(bangumi_data)
    await pikpak_manager.monitor_downloads()
    logging.info("自动化下载任务完成")


if __name__ == "__main__":
    asyncio.run(main())