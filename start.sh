#!/bin/bash
cd "$(dirname "$(readlink -f "$0")")"

if [ ! -x venv/bin/python ]; then
    echo "虚拟环境未创建，正在初始化..."
    python3 -m venv venv || { echo "创建 venv 失败"; exit 1; }
    venv/bin/pip install -r requirements.txt || { echo "安装依赖失败"; exit 1; }
fi

venv/bin/python app.py
