import json
import io
import os
import sqlite3
import tempfile
from pathlib import Path
from unittest.mock import patch, MagicMock

import pytest
from app import app, init_db, get_db, DB_PATH, _extract_number

TEST_DB = os.path.join(tempfile.gettempdir(), "anime_manager_test.db")


@pytest.fixture(autouse=True)
def setup_test_db(monkeypatch):
    monkeypatch.setattr("app.DB_PATH", TEST_DB)
    if os.path.exists(TEST_DB):
        os.remove(TEST_DB)
    init_db()
    yield
    if os.path.exists(TEST_DB):
        os.remove(TEST_DB)


@pytest.fixture
def client():
    app.config["TESTING"] = True
    return app.test_client()


def _insert_mapping(bangumi_id, title_cn, title_jp, folder_path, image_url="", rating=0, total_episodes=0, platform="", date="", summary="", tags=None):
    conn = get_db()
    tags_json = json.dumps(tags or [], ensure_ascii=False)
    conn.execute("""INSERT INTO mappings
        (bangumi_id, title_cn, title_jp, image_url, rating, total_episodes,
         platform, date, summary, tags_json, folder_path)
        VALUES (?,?,?,?,?,?,?,?,?,?,?)""",
        (bangumi_id, title_cn, title_jp, image_url, rating, total_episodes,
         platform, date, summary, tags_json, folder_path))
    conn.commit()
    conn.close()


def _last_id():
    conn = get_db()
    row = conn.execute("SELECT MAX(id) FROM mappings").fetchone()
    conn.close()
    return row[0]


# ─── DB tests ────────────────────────────────────────────

class TestDatabase:
    def test_init_creates_table(self):
        conn = get_db()
        tables = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='mappings'"
        ).fetchall()
        conn.close()
        assert len(tables) == 1

    def test_get_db(self):
        conn = get_db()
        assert isinstance(conn, sqlite3.Connection)
        conn.close()


# ─── Mappings CRUD ───────────────────────────────────────

class TestMappingsCRUD:
    def test_list_empty(self, client):
        resp = client.get("/api/mappings")
        assert resp.status_code == 200
        assert resp.json == []

    def test_list_with_data(self, client):
        _insert_mapping(1, "进击的巨人", "進撃の巨人", "/anime/a")
        _insert_mapping(2, "鬼灭之刃", "鬼滅の刃", "/anime/b")
        resp = client.get("/api/mappings")
        data = resp.json
        assert len(data) == 2
        assert data[0]["title_cn"] == "鬼灭之刃"
        assert data[1]["bangumi_id"] == 1

    def test_add_mapping_with_metadata(self, client):
        resp = client.post("/api/mappings", json={
            "bangumi_id": 42, "title_cn": "Test", "title_jp": "Test JP",
            "image_url": "https://example.com/img.jpg",
            "rating": 8.5, "total_episodes": 12, "platform": "TV",
            "date": "2023-01-01", "summary": "A test",
            "tags": ["action", "drama"], "folder_path": "/tmp/test",
        })
        assert resp.status_code == 201
        data = resp.json
        assert data["bangumi_id"] == 42
        assert data["rating"] == 8.5
        assert data["total_episodes"] == 12
        assert data["platform"] == "TV"
        assert "action" in data["tags_json"]
        assert data["tags_json"] == '["action", "drama"]'

    def test_add_duplicate_folder(self, client):
        _insert_mapping(1, "a", "b", "/dup")
        resp = client.post("/api/mappings", json={"bangumi_id": 1, "folder_path": "/dup"})
        assert resp.status_code == 409

    def test_add_missing_fields(self, client):
        resp = client.post("/api/mappings", json={})
        assert resp.status_code == 400

    def test_delete(self, client):
        _insert_mapping(1, "x", "y", "/del")
        mid = _last_id()
        resp = client.delete(f"/api/mappings/{mid}")
        assert resp.json["ok"]
        assert client.get("/api/mappings").json == []

    def test_update(self, client):
        _insert_mapping(1, "old", "old", "/old")
        mid = _last_id()
        resp = client.put(f"/api/mappings/{mid}", json={
            "bangumi_id": 99, "title_cn": "new", "title_jp": "new", "folder_path": "/new",
            "rating": 9.0, "total_episodes": 26, "tags": ["test"],
        })
        assert resp.json["bangumi_id"] == 99
        assert resp.json["rating"] == 9.0
        assert resp.json["total_episodes"] == 26

    def test_list_order(self, client):
        _insert_mapping(3, "C", "", "/c")
        _insert_mapping(2, "B", "", "/b")
        _insert_mapping(1, "A", "", "/a")
        data = client.get("/api/mappings").json
        assert data[0]["title_cn"] == "A"
        assert data[2]["title_cn"] == "C"


# ─── Bangumi API proxies ─────────────────────────────────

def _mock_resp(json_data, status=200):
    m = MagicMock()
    m.json.return_value = json_data
    m.status_code = status
    return m


class TestBangumiSearch:
    def test_empty_query(self, client):
        assert client.get("/api/bangumi/search?q=").json == []

    def test_success(self, client):
        with patch("app.requests.get", return_value=_mock_resp({"list": [
            {"id": 1, "name": "S;G", "name_cn": "石头门", "rating": {"score": 8.8},
             "images": {"common": "https://x"}, "summary": "desc"}]})):
            data = client.get("/api/bangumi/search?q=命运").json
            assert data[0]["name_cn"] == "石头门"
            assert data[0]["score"] == 8.8

    def test_empty_list(self, client):
        with patch("app.requests.get", return_value=_mock_resp({"list": []})):
            assert client.get("/api/bangumi/search?q=xxx").json == []

    def test_error(self, client):
        with patch("app.requests.get", side_effect=Exception("fail")):
            resp = client.get("/api/bangumi/search?q=x")
            assert resp.status_code == 500


class TestBangumiSubject:
    def test_success(self, client):
        fake = {"id": 1, "name": "Jp", "name_cn": "中文", "summary": "s", "images": {"common": "img"},
                "rating": {"score": 7.5}, "rank": 5, "total_episodes": 12,
                "tags": [{"name": "A"}, {"name": "B"}], "platform": "TV", "date": "2020", "nsfw": True}
        with patch("app.requests.get", return_value=_mock_resp(fake)):
            data = client.get("/api/bangumi/subject/1").json
            assert data["name_cn"] == "中文"
            assert data["tags"] == ["A", "B"]
            assert data["nsfw"] is True

    def test_minimal(self, client):
        with patch("app.requests.get", return_value=_mock_resp({"id": 1, "name": "X"})):
            data = client.get("/api/bangumi/subject/1").json
            assert data["rating"] == 0
            assert data["tags"] == []
            assert data["image"] == ""


class TestBangumiEpisodes:
    def test_success(self, client):
        fake = {"data": [{"id": 1, "name": "ep1", "name_cn": "集1", "sort": 1, "type": 0, "airdate": "2020-01-01"}]}
        with patch("app.requests.get", return_value=_mock_resp(fake)):
            data = client.get("/api/bangumi/episodes/1").json
            assert data[0]["sort"] == 1
            assert data[0]["name_cn"] == "集1"

    def test_empty(self, client):
        with patch("app.requests.get", return_value=_mock_resp({})):
            assert client.get("/api/bangumi/episodes/1").json == []


# ─── Episode number extraction ───────────────────────────

class TestExtractNumber:
    @pytest.mark.parametrize("name,expected", [
        ("[01] Episode.mkv", 1),
        ("MyShow_02_.mp4", 2),
        ("Anime -03-.mkv", 3),
        ("Show.04.720p.mp4", 4),
        ("第05话.mp4", 5),
        ("第6話.mkv", 6),
        ("E07.mkv", 7),
        ("Ep08.mp4", 8),
        ("Episode 09.mp4", 9),
        ("S01E10.mp4", 10),
        ("Show [01].mkv", 1),
        ("No Number.mp4", None),
        ("(2001).mkv", None),
    ])
    def test_extract(self, name, expected):
        assert _extract_number(name) == expected


# ─── Match endpoint ──────────────────────────────────────

class TestMatchEndpoint:
    def test_missing_params(self, client):
        assert client.get("/api/match").status_code == 400

    def test_match(self, client, tmp_path):
        (tmp_path / "ep01.mkv").touch()
        (tmp_path / "ep02.mkv").touch()
        fake_eps = {"data": [
            {"id": 1, "name": "一", "name_cn": "第一集", "sort": 1, "type": 0, "airdate": "2020-01-01"},
            {"id": 2, "name": "二", "name_cn": "第二集", "sort": 2, "type": 0, "airdate": "2020-01-08"},
            {"id": 3, "name": "三", "name_cn": "第三集", "sort": 3, "type": 0, "airdate": "2020-01-15"},
        ]}
        with patch("app.requests.get", return_value=_mock_resp(fake_eps)):
            resp = client.get(f"/api/match?bangumi_id=1&folder={tmp_path}")
            data = resp.json
            assert len(data) == 3
            assert data[0]["sort"] == 1
            assert data[0]["file"] is not None
            assert data[0]["file"]["name"] == "ep01.mkv"
            assert data[1]["file"]["name"] == "ep02.mkv"
            assert data[2]["file"] is None  # ep03 unmatched

    def test_unmatched_file(self, client, tmp_path):
        (tmp_path / "SP.mkv").touch()
        fake_eps = {"data": [{"id": 1, "name": "e1", "sort": 1, "type": 0, "airdate": ""}]}
        with patch("app.requests.get", return_value=_mock_resp(fake_eps)):
            data = client.get(f"/api/match?bangumi_id=1&folder={tmp_path}").json
            assert len(data) == 2
            assert data[0]["sort"] == 1
            assert data[0]["file"] is None
            assert data[1]["sort"] is None
            assert data[1]["file"]["name"] == "SP.mkv"


# ─── File scanning ───────────────────────────────────────

class TestScanDirectory:
    def test_missing_path(self, client):
        assert client.get("/api/scan?path=/no").status_code == 400

    def test_folders_and_files(self, client, tmp_path):
        (tmp_path / "Season1").mkdir()
        (tmp_path / "movie.mp4").touch()
        (tmp_path / "readme.txt").touch()
        data = client.get(f"/api/scan?path={tmp_path}").json
        names = [d["name"] for d in data]
        assert "Season1" in names
        assert "movie.mp4" in names
        assert "readme.txt" not in names

    def test_hidden_excluded(self, client, tmp_path):
        (tmp_path / ".hidden").mkdir()
        (tmp_path / ".secret.mkv").touch()
        data = client.get(f"/api/scan?path={tmp_path}").json
        assert data == []


# ─── Page routes ─────────────────────────────────────────

class TestPages:
    def test_index(self, client):
        resp = client.get("/")
        assert resp.status_code == 200
        assert "AniVault" in resp.text

    def test_index_with_data(self, client):
        _insert_mapping(1, "进击", "Shingeki", "/a", rating=8.5, total_episodes=87, image_url="https://x.jpg")
        resp = client.get("/")
        assert "进击" in resp.text
        assert "8.5" in resp.text

    def test_show_404(self, client):
        assert client.get("/show/999").status_code == 404

    def test_show_page(self, client):
        _insert_mapping(1, "巨人", "Shingeki", "/a", image_url="http://img.jpg", rating=8.5, platform="TV", summary="summary", tags=["热血"])
        mid = _last_id()
        resp = client.get(f"/show/{mid}")
        assert resp.status_code == 200
        assert "巨人" in resp.text
        assert "img.jpg" in resp.text
        assert "match-table" in resp.text

    def test_manage_page(self, client):
        resp = client.get("/manage")
        assert resp.status_code == 200
        assert "管理番剧关联" in resp.text


# ─── Export / Import ─────────────────────────────────────

class TestExportImport:
    def test_export_empty(self, client):
        resp = client.get("/api/export")
        assert resp.status_code == 200
        assert resp.json == []

    def test_export_with_data(self, client):
        _insert_mapping(1, "A", "B", "/a", rating=8.5, total_episodes=12)
        resp = client.get("/api/export")
        data = resp.json
        assert len(data) == 1
        assert data[0]["title_cn"] == "A"
        assert data[0]["rating"] == 8.5

    def test_import_adds_new(self, client):
        data = [{"bangumi_id": 1, "title_cn": "Imported", "folder_path": "/import/new"}]
        resp = client.post("/api/import", data={"file": (io.BytesIO(json.dumps(data).encode()), "test.json")})
        assert resp.json["added"] == 1
        mappings = client.get("/api/mappings").json
        assert mappings[0]["title_cn"] == "Imported"

    def test_import_skips_duplicate(self, client):
        _insert_mapping(1, "Existing", "", "/import/dup")
        data = [{"bangumi_id": 99, "title_cn": "Should Skip", "folder_path": "/import/dup"}]
        resp = client.post("/api/import", data={"file": (io.BytesIO(json.dumps(data).encode()), "test.json")})
        assert resp.json["skipped"] == 1
        assert resp.json["added"] == 0

    def test_import_no_file(self, client):
        resp = client.post("/api/import")
        assert resp.status_code == 400

    def test_import_bad_json(self, client):
        resp = client.post("/api/import", data={"file": (io.BytesIO(b"not json"), "test.json")})
        assert resp.status_code == 400

    def test_import_partial(self, client):
        data = [
            {"bangumi_id": 1, "folder_path": "/a"},
            {"bangumi_id": 2, "folder_path": "/b"},
        ]
        _insert_mapping(1, "dup", "", "/a")
        resp = client.post("/api/import", data={"file": (io.BytesIO(json.dumps(data).encode()), "test.json")})
        assert resp.json["added"] == 1
        assert resp.json["skipped"] == 1


# ─── Regex endpoint ──────────────────────────────────────

class TestRegexEndpoint:
    def test_update_regex(self, client):
        _insert_mapping(1, "X", "", "/r")
        mid = _last_id()
        resp = client.put(f"/api/mappings/{mid}/regex", json={"pattern": r"S\d+E(\d+)"})
        assert resp.json["regex_pattern"] == r"S\d+E(\d+)"

    def test_clear_regex(self, client):
        _insert_mapping(1, "X", "", "/rc")
        mid = _last_id()
        client.put(f"/api/mappings/{mid}/regex", json={"pattern": ""})
        resp = client.get("/api/mappings")
        assert resp.json[0]["regex_pattern"] == ""

    def test_regex_nonexistent(self, client):
        resp = client.put("/api/mappings/99999/regex", json={"pattern": "x"})
        assert resp.status_code == 404


# ─── Open endpoint ───────────────────────────────────────

class TestOpenEndpoint:
    def test_open_missing_file(self, client):
        resp = client.get("/api/open?path=/no/such")
        assert resp.status_code == 404

    def test_open_non_video(self, client, tmp_path):
        f = tmp_path / "readme.txt"
        f.touch()
        resp = client.get(f"/api/open?path={f}")
        assert resp.status_code == 400
        assert "不是视频" in resp.json["error"]

    def test_open_video(self, client, tmp_path):
        f = tmp_path / "test.mp4"
        f.touch()
        mock_popen = MagicMock()
        with patch("subprocess.Popen", return_value=mock_popen) as mock_p:
            resp = client.get(f"/api/open?path={f}")
            assert resp.status_code == 200
            mock_p.assert_called_once()


# ─── Extract number edge cases ───────────────────────────

class TestExtractNumberAdvanced:
    def test_custom_pattern(self):
        assert _extract_number("Show_第12話.mkv", r"第\s*(\d+)\s*[話话]") == 12

    def test_custom_pattern_no_group(self):
        # Pattern without capture group should not crash
        assert _extract_number("Show 01.mkv", r"\d+") is None

    def test_max_episode_guard(self):
        # 1080 is > 200, should be rejected by fallback
        assert _extract_number("Anime - 1080.mkv") is None

    def test_year_rejected(self):
        # Year 2026 > 200, should be rejected
        assert _extract_number("Anime 2026 S01E01.mkv") == 1
