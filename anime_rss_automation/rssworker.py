from auto_anime_dl.config import RSS_URLS, BANGUMI_JSON_PATH, BANGUMI_WITH_COVER_JSON_PATH
from auto_anime_dl.rss import create_ssl_context, get_rss_feed, extract_rss_data, enhance_bangumi_data
from auto_anime_dl.utils import write_json


if __name__ == "__main__":
    create_ssl_context()
    merged: dict[str, list[dict]] = {}
    print(f"开始处理 {len(RSS_URLS)} 个RSS源...")
    for name, url in RSS_URLS.items():
        print(f"正在处理: {name} ({url})")
        feed_data = get_rss_feed(url)
        if not feed_data:
            print("  处理失败，跳过该源")
            continue
        extracted = extract_rss_data(feed_data)
        if not extracted:
            print("  未提取到有效数据")
            continue
        for anime_title, episodes in extracted.items():
            merged.setdefault(anime_title, []).extend(episodes)

    if write_json({"Bangumi": merged}, BANGUMI_JSON_PATH):
        print(f"\n✓ 原始数据已成功保存到: {BANGUMI_JSON_PATH}")
        anime_count = len(merged)
        episode_count = sum(len(episodes) for episodes in merged.values())
        print(f"共收集了 {anime_count} 部动画，{episode_count} 个剧集")
    else:
        print("\n✗ 保存原始文件失败")
        raise SystemExit(1)

    enhanced_data = enhance_bangumi_data({"Bangumi": merged})
    if write_json({"Bangumi": enhanced_data}, BANGUMI_WITH_COVER_JSON_PATH):
        print(f"\n✓ 增强数据已成功保存到: {BANGUMI_WITH_COVER_JSON_PATH}")
        print("数据增强完成")
    else:
        print("\n✗ 保存增强文件失败")