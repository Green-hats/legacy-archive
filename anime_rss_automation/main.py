import os
from typing import Any, Dict

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from fastapi.staticfiles import StaticFiles

from auto_anime_dl.config import (
    BANGUMI_JSON_PATH,
    BANGUMI_WITH_COVER_JSON_PATH,
    BANGUMI_WITH_PLAY_URL_PATH,
)
from auto_anime_dl.utils import read_json
from auto_anime_dl.scheduler import run_scheduler, run_once
from auto_anime_dl.state import load_processed_keys
from auto_anime_dl.config import STATE_PATH, SCHEDULE_INTERVAL_SECONDS
import asyncio
import logging


app = FastAPI()

BASE_DIR = os.path.dirname(os.path.abspath(__file__))


def _load_data_or_404(path: str) -> Dict[str, Any]:
    data = read_json(path)
    if not data:
        raise HTTPException(status_code=404, detail="数据未找到或为空")
    return data


@app.get("/api/bangumi")
def get_bangumi():
    return JSONResponse(content=_load_data_or_404(BANGUMI_JSON_PATH))


@app.get("/api/bangumi_with_cover")
def get_bangumi_with_cover():
    return JSONResponse(content=_load_data_or_404(BANGUMI_WITH_COVER_JSON_PATH))


@app.get("/api/bangumi_with_play_url")
def get_bangumi_with_play_url():
    return JSONResponse(content=_load_data_or_404(BANGUMI_WITH_PLAY_URL_PATH))


# 挂载 frontend 目录，自动支持 index.html（需在所有 API 路由之后再挂载，避免遮蔽）


@app.on_event("startup")
async def _startup_background_tasks():
    # 在后台启动RSS调度器
    asyncio.create_task(run_scheduler())


@app.get("/api/scheduler/status")
def scheduler_status():
    processed = load_processed_keys(STATE_PATH)
    return JSONResponse(content={
        "interval_seconds": SCHEDULE_INTERVAL_SECONDS,
        "processed_count": len(processed),
    })


@app.post("/api/scheduler/run_once")
async def scheduler_run_once():
    result = await run_once()
    return JSONResponse(content=result)


@app.get("/api/scheduler/run_once")
async def scheduler_run_once_get():
    # 便于浏览器直接点击触发
    return await scheduler_run_once()

# 最后挂载静态目录，避免与 /api/* 路由冲突
app.mount("/", StaticFiles(directory=os.path.join(BASE_DIR, "frontend"), html=True), name="frontend")
