import sqlite3
import json
import os
import re
import logging
import subprocess
from urllib.parse import quote
from flask import Flask, jsonify, request, render_template, Response
import requests

app = Flask(__name__)
logging.basicConfig(level=logging.WARNING, format="[%(levelname)s] %(message)s")
DB_PATH = os.path.join(os.path.dirname(__file__), "data", "mappings.db")

BANGUMI_API = "https://api.bgm.tv"
HEADERS = {"User-Agent": "AniVault/1.0 (local tool)"}

VIDEO_EXTENSIONS = {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".rmvb"}


def get_db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn


def init_db():
    os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
    conn = get_db()
    conn.execute("""CREATE TABLE IF NOT EXISTS mappings (
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        bangumi_id      INTEGER NOT NULL,
        title_cn        TEXT DEFAULT '',
        title_jp        TEXT DEFAULT '',
        image_url       TEXT DEFAULT '',
        rating          REAL DEFAULT 0,
        total_episodes  INTEGER DEFAULT 0,
        platform        TEXT DEFAULT '',
        date            TEXT DEFAULT '',
        summary         TEXT DEFAULT '',
        tags_json       TEXT DEFAULT '[]',
        regex_pattern   TEXT DEFAULT '',
        episodes_json   TEXT DEFAULT '',
        episodes_at     TEXT DEFAULT '',
        folder_path     TEXT NOT NULL UNIQUE,
        created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )""")
    # Add missing columns for schema upgrade from v1
    for col, coltype in [
        ("image_url", "TEXT DEFAULT ''"),
        ("rating", "REAL DEFAULT 0"),
        ("total_episodes", "INTEGER DEFAULT 0"),
        ("platform", "TEXT DEFAULT ''"),
        ("date", "TEXT DEFAULT ''"),
        ("summary", "TEXT DEFAULT ''"),
        ("tags_json", "TEXT DEFAULT '[]'"),
        ("regex_pattern", "TEXT DEFAULT ''"),
        ("episodes_json", "TEXT DEFAULT ''"),
        ("episodes_at", "TEXT DEFAULT ''"),
    ]:
        try:
            conn.execute(f"ALTER TABLE mappings ADD COLUMN {col} {coltype}")
        except sqlite3.OperationalError:
            pass
    conn.commit()
    conn.close()


# ─── Pages ───────────────────────────────────────────────

@app.route("/")
def index():
    conn = get_db()
    rows = conn.execute("SELECT * FROM mappings ORDER BY id DESC").fetchall()
    conn.close()
    return render_template("index.html", mappings=[dict(r) for r in rows])


@app.route("/show/<int:mapping_id>")
def show(mapping_id):
    conn = get_db()
    row = conn.execute("SELECT * FROM mappings WHERE id = ?", (mapping_id,)).fetchone()
    conn.close()
    if not row:
        return "Not Found", 404
    return render_template("show.html", m=dict(row))


@app.route("/manage")
def manage():
    default_path = os.path.expanduser("~")
    return render_template("manage.html", default_path=default_path)


# ─── Bangumi API proxies ─────────────────────────────────

@app.route("/api/bangumi/search")
def bangumi_search():
    q = request.args.get("q", "")
    if not q:
        return jsonify([])
    try:
        resp = requests.get(
            f"{BANGUMI_API}/search/subject/{quote(q)}",
            params={"type": 2, "max_results": 10},
            headers=HEADERS, timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        if isinstance(data, dict) and "list" in data:
            results = [{
                "id": item["id"],
                "name": item.get("name", ""),
                "name_cn": item.get("name_cn", ""),
                "score": item.get("rating", {}).get("score", 0) if item.get("rating") else 0,
                "image": item.get("images", {}).get("common", ""),
                "summary": item.get("summary", ""),
            } for item in data["list"]]
            return jsonify(results)
        return jsonify([])
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/api/bangumi/subject/<int:subject_id>")
def bangumi_subject(subject_id):
    try:
        resp = requests.get(f"{BANGUMI_API}/v0/subjects/{subject_id}", headers=HEADERS, timeout=10)
        resp.raise_for_status()
        data = resp.json()
        result = {
            "id": data["id"],
            "name": data.get("name", ""),
            "name_cn": data.get("name_cn", ""),
            "summary": data.get("summary", ""),
            "image": data.get("images", {}).get("common", "") if data.get("images") else "",
            "rating": data.get("rating", {}).get("score", 0) if data.get("rating") else 0,
            "rank": data.get("rank", 0),
            "total_episodes": data.get("total_episodes", 0),
            "tags": [t["name"] for t in data.get("tags", [])[:10]],
            "platform": data.get("platform", ""),
            "date": data.get("date", ""),
            "nsfw": data.get("nsfw", False),
        }
        return jsonify(result)
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/api/bangumi/episodes/<int:subject_id>")
def bangumi_episodes(subject_id):
    try:
        params = {"subject_id": subject_id, "limit": 200, "offset": 0}
        resp = requests.get(f"{BANGUMI_API}/v0/episodes", params=params, headers=HEADERS, timeout=10)
        resp.raise_for_status()
        data = resp.json()
        episodes = []
        if isinstance(data, dict) and "data" in data:
            for ep in data["data"]:
                episodes.append({
                    "id": ep["id"],
                    "name": ep.get("name", ""),
                    "name_cn": ep.get("name_cn", ""),
                    "sort": ep.get("sort", 0),
                    "type": ep.get("type", 0),
                    "airdate": ep.get("airdate", ""),
                })
        return jsonify(episodes)
    except Exception as e:
        return jsonify({"error": str(e)}), 500


# ─── File scanning ───────────────────────────────────────

@app.route("/api/scan")
def scan_directory():
    path = request.args.get("path", "")
    if not path or not os.path.exists(path):
        return jsonify({"error": "路径不存在"}), 400
    result = []
    try:
        for entry in sorted(os.listdir(path)):
            if entry.startswith("."):
                continue
            full = os.path.join(path, entry)
            if os.path.isdir(full):
                result.append({"name": entry, "path": full, "type": "folder"})
        for entry in sorted(os.listdir(path)):
            if entry.startswith("."):
                continue
            full = os.path.join(path, entry)
            ext = os.path.splitext(entry)[1].lower()
            if os.path.isfile(full) and ext in VIDEO_EXTENSIONS:
                result.append({"name": entry, "path": full, "type": "file"})
        return jsonify(result)
    except Exception as e:
        return jsonify({"error": str(e)}), 500


# ─── Video listing ────────────────────────────────────────

def _list_video_files(root):
    files = []
    try:
        for entry in sorted(os.listdir(root)):
            if entry.startswith("."):
                continue
            full = os.path.join(root, entry)
            if os.path.isdir(full):
                files.extend(_list_video_files(full))
            else:
                ext = os.path.splitext(entry)[1].lower()
                if ext in VIDEO_EXTENSIONS:
                    files.append({"name": entry, "path": full})
    except Exception:
        pass
    return files


@app.route("/api/videos")
def list_videos():
    path = request.args.get("path", "")
    if not path or not os.path.exists(path):
        return jsonify({"error": "路径不存在"}), 400
    files = _list_video_files(path)
    files.sort(key=lambda f: _extract_number(f["name"]) or 9999)
    return jsonify(files)


@app.route("/api/open")
def mpv_open():
    path = request.args.get("path", "")
    if not path or not os.path.isfile(path):
        return jsonify({"error": "文件不存在"}), 404
    if os.path.splitext(path)[1].lower() not in VIDEO_EXTENSIONS:
        return jsonify({"error": "不是视频文件"}), 400
    try:
        subprocess.Popen(["mpv", path], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                         start_new_session=True)
        return jsonify({"ok": True})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/api/playlist")
def playlist():
    path = request.args.get("path", "")
    if not path or not os.path.isfile(path):
        return jsonify({"error": "文件不存在"}), 404
    m3u = f"#EXTM3U\n#EXTINF:-1,{os.path.basename(path)}\n{path}\n"
    return Response(m3u, mimetype="audio/x-mpegurl",
                    headers={"Content-Disposition": f'inline; filename="{os.path.basename(path)}.m3u"'})

# ─── Episode number extraction ───────────────────────────

def _extract_number(name, custom_pattern=None):
    if custom_pattern:
        try:
            m = re.search(custom_pattern, name, re.IGNORECASE)
            if m:
                return int(m.group(1))
        except (IndexError, re.error):
            return None
    patterns = [
        r'[Ss]\d+[Ee]0*(\d+)',
        r'\bE0*(\d+)\b',
        r'\bEp(?:isode)?\s*0*(\d+)',
        r'第\s*0*(\d+)\s*[話话集]',
        r'\[0*(\d+)\]',
        r'[_\-. ]0*(\d+)[_\-. ]',
        r'[ _\-.](\d+)[ _\-.]',
    ]
    for p in patterns:
        m = re.search(p, name, re.IGNORECASE)
        if m:
            n = int(m.group(1))
            return n if n <= 200 else None
    return None


@app.route("/api/match")
def match_episodes():
    bangumi_id = request.args.get("bangumi_id", "")
    folder = request.args.get("folder", "")
    mapping_id = request.args.get("mapping_id", "")
    force_refresh = request.args.get("refresh", "0") == "1"

    if not bangumi_id or not folder:
        return jsonify({"error": "缺少参数"}), 400

    custom_pattern = request.args.get("regex", "")
    if not custom_pattern and mapping_id:
        conn = get_db()
        row = conn.execute("SELECT regex_pattern FROM mappings WHERE id = ?", (int(mapping_id),)).fetchone()
        conn.close()
        if row and row["regex_pattern"]:
            custom_pattern = row["regex_pattern"]

    bangumi_eps = []
    if mapping_id and not force_refresh:
        conn = get_db()
        row = conn.execute("SELECT episodes_json FROM mappings WHERE id = ?", (int(mapping_id),)).fetchone()
        conn.close()
        if row and row["episodes_json"]:
            try:
                bangumi_eps = json.loads(row["episodes_json"])
            except (json.JSONDecodeError, TypeError):
                pass

    if not bangumi_eps:
        try:
            params = {"subject_id": int(bangumi_id), "limit": 200, "offset": 0}
            resp = requests.get(f"{BANGUMI_API}/v0/episodes", params=params, headers=HEADERS, timeout=10)
            resp.raise_for_status()
            data = resp.json()
            if isinstance(data, dict) and "data" in data:
                for ep in data["data"]:
                    bangumi_eps.append({
                        "id": ep["id"], "name": ep.get("name", ""),
                        "name_cn": ep.get("name_cn", ""), "sort": ep.get("sort", 0),
                        "airdate": ep.get("airdate", ""),
                    })
            if mapping_id and bangumi_eps:
                conn = get_db()
                conn.execute("UPDATE mappings SET episodes_json = ?, episodes_at = datetime('now') WHERE id = ?",
                             (json.dumps(bangumi_eps, ensure_ascii=False), int(mapping_id)))
                conn.commit()
                conn.close()
        except Exception as e:
            logging.warning("Bangumi episodes fetch failed: %s", e)

    local_files = _list_video_files(folder)
    local_files.sort(key=lambda f: _extract_number(f["name"], custom_pattern) or 9999)

    matched = set()
    rows = []

    for ep in bangumi_eps:
        file_info = None
        for f in local_files:
            if f["path"] in matched:
                continue
            num = _extract_number(f["name"], custom_pattern)
            if num is not None and num == ep["sort"]:
                file_info = f
                matched.add(f["path"])
                break
        rows.append({
            "sort": ep["sort"],
            "bangumi_name": ep.get("name_cn") or ep.get("name") or "",
            "bangumi_raw": ep.get("name", ""),
            "airdate": ep.get("airdate", ""),
            "file": file_info,
        })

    for f in local_files:
        if f["path"] not in matched:
            rows.append({
                "sort": None,
                "bangumi_name": "",
                "bangumi_raw": "",
                "airdate": "",
                "file": f,
            })

    return jsonify(rows)


# ─── Mappings CRUD ───────────────────────────────────────

@app.route("/api/mappings")
def list_mappings():
    conn = get_db()
    rows = conn.execute("SELECT * FROM mappings ORDER BY id DESC").fetchall()
    conn.close()
    return jsonify([dict(r) for r in rows])


@app.route("/api/mappings", methods=["POST"])
def add_mapping():
    data = request.json or {}
    bangumi_id = data.get("bangumi_id")
    folder_path = data.get("folder_path")
    if not bangumi_id or not folder_path:
        return jsonify({"error": "缺少必填字段"}), 400

    title_cn = data.get("title_cn", "")
    title_jp = data.get("title_jp", "")
    image_url = data.get("image_url", "")
    rating = data.get("rating", 0)
    total_episodes = data.get("total_episodes", 0)
    platform = data.get("platform", "")
    date = data.get("date", "")
    summary = data.get("summary", "")
    tags_json = json.dumps(data.get("tags", []), ensure_ascii=False)
    regex_pattern = data.get("regex_pattern", "")

    conn = get_db()
    try:
        conn.execute("""INSERT INTO mappings
            (bangumi_id, title_cn, title_jp, image_url, rating, total_episodes,
             platform, date, summary, tags_json, regex_pattern, folder_path)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?)""",
            (bangumi_id, title_cn, title_jp, image_url, rating, total_episodes,
             platform, date, summary, tags_json, regex_pattern, folder_path))
        conn.commit()
        new_id = conn.execute("SELECT last_insert_rowid()").fetchone()[0]
        row = conn.execute("SELECT * FROM mappings WHERE id = ?", (new_id,)).fetchone()
        conn.close()
        return jsonify(dict(row)), 201
    except sqlite3.IntegrityError:
        conn.close()
        return jsonify({"error": "该路径已关联"}), 409


@app.route("/api/mappings/<int:mapping_id>", methods=["DELETE"])
def delete_mapping(mapping_id):
    conn = get_db()
    cursor = conn.execute("DELETE FROM mappings WHERE id = ?", (mapping_id,))
    conn.commit()
    conn.close()
    if cursor.rowcount == 0:
        return jsonify({"error": "关联不存在"}), 404
    return jsonify({"ok": True})


@app.route("/api/mappings/<int:mapping_id>", methods=["PUT"])
def update_mapping(mapping_id):
    data = request.json or {}
    conn = get_db()
    conn.execute("""UPDATE mappings SET
        bangumi_id=?, title_cn=?, title_jp=?, image_url=?, rating=?,
        total_episodes=?, platform=?, date=?, summary=?, tags_json=?, regex_pattern=?, folder_path=?
        WHERE id=?""",
        (data.get("bangumi_id"), data.get("title_cn", ""), data.get("title_jp", ""),
         data.get("image_url", ""), data.get("rating", 0), data.get("total_episodes", 0),
         data.get("platform", ""), data.get("date", ""), data.get("summary", ""),
         json.dumps(data.get("tags", []), ensure_ascii=False),
         data.get("regex_pattern", ""), data.get("folder_path"),
         mapping_id))
    conn.commit()
    row = conn.execute("SELECT * FROM mappings WHERE id = ?", (mapping_id,)).fetchone()
    conn.close()
    return jsonify(dict(row))


@app.route("/api/mappings/<int:mapping_id>/regex", methods=["PUT"])
def update_regex(mapping_id):
    data = request.json or {}
    pattern = data.get("pattern", "")
    conn = get_db()
    conn.execute("UPDATE mappings SET regex_pattern = ? WHERE id = ?", (pattern, mapping_id))
    conn.commit()
    row = conn.execute("SELECT regex_pattern FROM mappings WHERE id = ?", (mapping_id,)).fetchone()
    conn.close()
    if not row:
        return jsonify({"error": "关联不存在"}), 404
    return jsonify({"regex_pattern": row["regex_pattern"] or ""})


# ─── Export / Import ─────────────────────────────────────

@app.route("/api/export")
def export_data():
    conn = get_db()
    rows = conn.execute("SELECT * FROM mappings ORDER BY id DESC").fetchall()
    conn.close()
    data = [dict(r) for r in rows]
    return Response(json.dumps(data, ensure_ascii=False, indent=2),
                    mimetype="application/json",
                    headers={"Content-Disposition": "attachment; filename=anivault-backup.json"})


@app.route("/api/import", methods=["POST"])
def import_data():
    file = request.files.get("file")
    if not file:
        return jsonify({"error": "未上传文件"}), 400
    try:
        data = json.loads(file.read())
    except (json.JSONDecodeError, Exception):
        return jsonify({"error": "JSON 格式错误"}), 400
    if not isinstance(data, list):
        return jsonify({"error": "JSON 格式错误"}), 400

    conn = get_db()
    added = 0
    skipped = 0
    for item in data:
        if not item.get("folder_path") or not item.get("bangumi_id"):
            continue
        existing = conn.execute("SELECT id FROM mappings WHERE folder_path = ?",
                                (item["folder_path"],)).fetchone()
        if existing:
            skipped += 1
            continue
        try:
            conn.execute("""INSERT INTO mappings
                (bangumi_id, title_cn, title_jp, image_url, rating, total_episodes,
                 platform, date, summary, tags_json, regex_pattern,
                 episodes_json, episodes_at, folder_path)
                VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
                (item.get("bangumi_id"), item.get("title_cn", ""), item.get("title_jp", ""),
                 item.get("image_url", ""), item.get("rating", 0), item.get("total_episodes", 0),
                 item.get("platform", ""), item.get("date", ""), item.get("summary", ""),
                 item.get("tags_json", "[]"), item.get("regex_pattern", ""),
                 item.get("episodes_json", ""), item.get("episodes_at", ""),
                 item["folder_path"]))
            added += 1
        except Exception:
            skipped += 1
    conn.commit()
    conn.close()
    return jsonify({"added": added, "skipped": skipped, "total": len(data)})


@app.template_filter("from_json")
def from_json(value):
    if not value:
        return []
    try:
        return json.loads(value)
    except (json.JSONDecodeError, TypeError):
        return []


if __name__ == "__main__":
    init_db()
    print("AniVault running at http://127.0.0.1:5000")
    app.run(debug=True, host="127.0.0.1", port=5000, use_reloader=False)
