// detail.js
// 详情页脚本，负责渲染详情内容

const detailTitle = document.getElementById('detail-title');
const detailPoster = document.getElementById('detail-poster');
const detailMeta = document.getElementById('detail-meta');
const detailDesc = document.getElementById('detail-desc');
const episodeList = document.getElementById('episode-list');
const magnetLink = document.getElementById('magnet-link');
const backBtn = document.getElementById('back-btn');

// 获取 query 参数
function getQueryParam(name) {
    const url = new URL(window.location.href);
    return url.searchParams.get(name);
}

async function loadAnimeData() {
    try {
        const response = await fetch('/api/bangumi_with_play_url');
        if (!response.ok) throw new Error('加载数据失败');
        const data = await response.json();
        return data.Bangumi;
    } catch (error) {
        detailTitle.textContent = '加载数据失败';
        return {};
    }
}


function renderDetail(title, anime) {
    detailTitle.textContent = title;
    detailPoster.innerHTML = `<img src="${anime.cover_info?.cover_url || ''}" alt="${title}"/>`;
    detailMeta.innerHTML = `<span>剧集: ${anime.episodes.length}</span>`;
    detailDesc.textContent = '这部动漫讲述了精彩的故事内容...';
    episodeList.innerHTML = '';
    const video = document.getElementById('flv-video');
    const placeholder = document.getElementById('video-placeholder');
    const openIINA = document.getElementById('open-iina');
    const openPotPlayer = document.getElementById('open-potplayer');
    let flvPlayer = null;
    let currentMagnet = '';
    let currentVideoUrl = '';
    const episodes = anime.episodes.slice().reverse();
    function setExternalPlayerHandlers() {
        if (openIINA) {
            openIINA.onclick = function () {
                let url = currentVideoUrl || currentMagnet;
                if (url) {
                    window.location.href = 'iina://weblink?url=' + encodeURIComponent(url);
                } else {
                    alert('请先选择剧集');
                }
            };
        }
        if (openPotPlayer) {
            openPotPlayer.onclick = function () {
                let url = currentVideoUrl || currentMagnet;
                if (url) {
                    window.location.href = 'potplayer://url/' + encodeURIComponent(url);
                } else {
                    alert('请先选择剧集');
                }
            };
        }
    }
    episodes.forEach((ep, i) => {
        const btn = document.createElement('button');
        btn.className = 'episode-btn';
        btn.innerHTML = `<span>第${i + 1}集</span>`;
        btn.onclick = () => {
            document.querySelectorAll('.episode-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            if (ep.bt_url) {
                magnetLink.innerHTML = `<button id="copy-magnet-btn" class="btn btn-secondary" style="padding:4px 12px;font-size:0.95em;">点击复制磁力链接</button>`;
                const copyBtn = document.getElementById('copy-magnet-btn');
                copyBtn.onclick = function () {
                    navigator.clipboard.writeText(ep.bt_url).then(() => {
                        copyBtn.textContent = '已复制!';
                        setTimeout(() => { copyBtn.textContent = '点击复制磁力链接'; }, 1200);
                    });
                };
            } else {
                magnetLink.textContent = '';
            }
            currentMagnet = ep.bt_url || '';
            currentVideoUrl = ep.play_url || '';
            setExternalPlayerHandlers();
            // flv.js 播放器逻辑
            if (flvPlayer) {
                flvPlayer.destroy();
                flvPlayer = null;
            }
            video.style.display = 'block';
            placeholder.style.display = 'none';
            video.pause();
            video.src = '';
            if (window.flvjs && window.flvjs.isSupported()) {
                flvPlayer = window.flvjs.createPlayer({
                    type: 'mkv',
                    url: currentVideoUrl // 这里用 ep.play_url
                });
                flvPlayer.attachMediaElement(video);
                flvPlayer.load();
                flvPlayer.play();
            }
        };
        episodeList.appendChild(btn);
    });
    // 初始化时禁用外部播放器按钮
    setExternalPlayerHandlers();
}

backBtn.addEventListener('click', () => {
    window.location.href = 'index.html';
});

// 页面初始化
(async function initDetail() {
    const title = getQueryParam('title');
    if (!title) {
        detailTitle.textContent = '未指定动漫';
        return;
    }
    const data = await loadAnimeData();
    const anime = data[title];
    if (!anime) {
        detailTitle.textContent = '未找到该动漫';
        return;
    }
    renderDetail(title, anime);
})();
