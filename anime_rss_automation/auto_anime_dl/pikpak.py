import asyncio
import logging
import re
import time
from typing import Dict, Optional, Any

import aiohttp
import httpx
from pikpakapi import PikPakApi


def sanitize_folder_name(name: str) -> str:
    sanitized = re.sub(r'[\\/:*?"<>|]', ' ', name)
    sanitized = sanitized.strip()
    if len(sanitized) > 100:
        sanitized = sanitized[:100] + "..."
    return sanitized


class PikPakManager:
    def __init__(self, username: str, password: str, root_folder_id: Optional[str] = None):
        self.api = PikPakApi(
            username=username,
            password=password,
            httpx_client_args={
                "transport": httpx.AsyncHTTPTransport(retries=3),
            },
        )
        self.root_folder_id = root_folder_id

    async def login(self) -> bool:
        try:
            await self.api.login()
            logging.info("登录PikPak成功")
            return True
        except Exception as exc:
            logging.error("登录失败: %s", exc)
            return False

    async def setup_root_folder(self, root_name: str = "Bangumi") -> bool:
        try:
            if not self.root_folder_id:
                logging.error("未提供 root_folder_id")
                return False
            await self.api.file_list(parent_id=self.root_folder_id)
            logging.info("根文件夹 '%s' 验证成功 (ID: %s)", root_name, self.root_folder_id)
            return True
        except Exception as exc:
            logging.error("根文件夹验证失败: %s", exc)
            return False

    async def create_folder(self, name: str, parent_id: str) -> Optional[str]:
        sanitized_name = sanitize_folder_name(name)
        try:
            folder_info = await self.api.file_list(parent_id=parent_id)
            for item in folder_info.get("files", []):
                if item.get("name") == sanitized_name and item.get("kind") == "drive#folder":
                    return item.get("id")
            response = await self.api.create_folder(name=sanitized_name, parent_id=parent_id)
            if response and response.get("file") and response["file"].get("id"):
                return response["file"]["id"]
            return None
        except Exception as exc:
            logging.error("创建文件夹 '%s' 失败: %s", sanitized_name, exc)
            return None

    async def add_download_task(self, bt_url: str, parent_id: str) -> bool:
        try:
            result = await self.api.offline_download(bt_url, parent_id=parent_id)
            if result and result.get("task") and result["task"].get("status") in ["queued", "downloading"]:
                return True
            return False
        except Exception as exc:
            logging.error("添加下载任务失败: %s", exc)
            return False

    async def get_tasks_status(self) -> Dict[str, Any]:
        try:
            return await self.api.offline_list()
        except Exception as exc:
            logging.error("获取任务状态失败: %s", exc)
            return {}

    async def process_bangumi_data(self, bangumi_data: Dict[str, Any]):
        if not self.root_folder_id:
            logging.error("根文件夹未设置")
            return
        for anime_title, anime_data in bangumi_data.items():
            anime_folder_id = await self.create_folder(anime_title, self.root_folder_id)
            if not anime_folder_id:
                continue
            for episode in anime_data.get("episodes", []):
                bt_url = episode.get("bt_url")
                if not bt_url:
                    continue
                await self.add_download_task(bt_url, anime_folder_id)
                await asyncio.sleep(1)

    async def monitor_downloads(self, interval: int = 300):
        while True:
            tasks = await self.get_tasks_status()
            tasks_list = tasks.get("tasks", [])
            if not tasks_list:
                await asyncio.sleep(interval)
                continue
            status_count = {"queued": 0, "downloading": 0, "completed": 0, "error": 0}
            for task in tasks_list:
                status = task.get("status", "unknown")
                if status in status_count:
                    status_count[status] += 1
            if status_count["queued"] == 0 and status_count["downloading"] == 0:
                break
            await asyncio.sleep(interval)


class PikPakScanner:
    def __init__(self, username: str, password: str):
        self.api = PikPakApi(username=username, password=password)
        self.folder_structure: Dict[str, Any] = {}
        self.file_cache: Dict[str, str] = {}
        self.client = aiohttp.ClientSession(connector=aiohttp.TCPConnector(ssl=False))

    async def login(self) -> bool:
        try:
            await self.api.login()
            return True
        except Exception as exc:
            logging.error("登录失败: %s", exc)
            return False

    async def close(self):
        await self.client.close()

    async def scan_folder_structure(self, root_folder_id: str):
        self.folder_structure = await self._scan_folder_recursive(root_folder_id)
        return self.folder_structure

    async def _scan_folder_recursive(self, folder_id: str, path: str = "") -> Dict[str, Any]:
        try:
            folder_info = await self.api.file_list(parent_id=folder_id)
        except Exception as exc:
            logging.error("获取文件夹 %s 内容失败: %s", folder_id, exc)
            return {"id": folder_id, "path": path, "files": [], "subfolders": {}}

        folder_data: Dict[str, Any] = {"id": folder_id, "path": path, "files": [], "subfolders": {}}
        for item in folder_info.get("files", []):
            if item.get("kind") == "drive#file":
                folder_data["files"].append({
                    "id": item.get("id"),
                    "name": item.get("name"),
                    "size": item.get("size"),
                })
        for item in folder_info.get("files", []):
            if item.get("kind") == "drive#folder":
                folder_name = item.get("name")
                folder_path = f"{path}/{folder_name}" if path else folder_name
                subfolder_id = item.get("id")
                folder_data["subfolders"][folder_name] = await self._scan_folder_recursive(subfolder_id, folder_path)
        return folder_data

    @staticmethod
    def normalize_name(name: str) -> str:
        normalized = re.sub(r'[^\w\s\u4e00-\u9fff]', ' ', name)
        normalized = re.sub(r'\s+', ' ', normalized).strip().lower()
        return normalized

    @staticmethod
    def extract_episode(text: str) -> Optional[int]:
        ep_match = re.search(r"E(\d+)", text, re.IGNORECASE)
        if ep_match:
            return int(ep_match.group(1))
        num_match = re.search(r"[-]\s*(\d+)\s", text)
        if num_match:
            return int(num_match.group(1))
        cn_match = re.search(r"第(\d+)[话集]", text)
        if cn_match:
            return int(cn_match.group(1))
        pure_match = re.search(r"\b(\d{2})\b", text)
        if pure_match:
            return int(pure_match.group(1))
        return None

    async def get_file_info(self, file_id: str) -> Dict[str, Any]:
        url = f"https://api-drive.mypikpak.com/drive/v1/files/{file_id}"
        headers = {"Authorization": f"Bearer {self.api.access_token}", "Content-Type": "application/json"}
        try:
            async with self.client.get(url, headers=headers) as response:
                response.raise_for_status()
                return await response.json()
        except Exception as exc:
            logging.error("获取文件信息失败: %s", exc)
            return {}

    async def get_file_play_url(self, file_id: str) -> str:
        if file_id in self.file_cache:
            return self.file_cache[file_id]
        file_info = await self.get_file_info(file_id)
        play_url = ""
        if "medias" in file_info and len(file_info["medias"]) > 0:
            media = file_info["medias"][0]
            if "link" in media and "url" in media["link"]:
                play_url = media["link"]["url"]
        if not play_url:
            play_url = file_info.get("web_content_link", "") or file_info.get("medialink", "")
        self.file_cache[file_id] = play_url
        return play_url

    async def find_episode_in_folder(self, folder: Dict[str, Any], episode_num: int) -> Optional[Dict[str, Any]]:
        for file in folder.get("files", []):
            file_episode = self.extract_episode(file["name"])
            if file_episode == episode_num:
                play_url = await self.get_file_play_url(file["id"])
                if play_url:
                    file["play_url"] = play_url
                    return file
        return None

    async def find_episode_by_anime(self, folder_structure: Dict[str, Any], anime_name: str, episode_num: int) -> Optional[Dict[str, Any]]:
        normalized_anime = self.normalize_name(anime_name)
        for folder_name, folder_data in folder_structure.get("subfolders", {}).items():
            if self.normalize_name(folder_name) == normalized_anime:
                result = await self.find_episode_in_folder(folder_data, episode_num)
                if result:
                    return result
                for sub_name, sub_folder in folder_data.get("subfolders", {}).items():
                    if self.normalize_name(sub_name) == normalized_anime:
                        result = await self.find_episode_in_folder(sub_folder, episode_num)
                        if result:
                            return result
        return None


