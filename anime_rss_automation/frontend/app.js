
const animeList = document.getElementById('anime-list');
const loadingIndicator = document.getElementById('loading-indicator');

async function loadAnimeData() {
    try {
        const response = await fetch('/api/bangumi_with_play_url');
        if (!response.ok) throw new Error('加载数据失败');
        const data = await response.json();
        return data.Bangumi;
    } catch (error) {
        console.error('加载JSON数据时出错:', error);
        loadingIndicator.innerHTML = `<p>加载数据失败</p>`;
        return {};
    }
}


function renderAnimeList(animes) {
    animeList.innerHTML = '';
    Object.keys(animes).forEach((title, index) => {
        const anime = animes[title];
        const episodes = anime.episodes || [];
        const card = document.createElement('div');
        card.className = 'anime-card';
        card.innerHTML = `
            <div class="anime-cover">
                <img src="${anime.cover_info?.cover_url || ''}" alt="${title}" />
                <div class="anime-episodes">${episodes.length}集</div>
            </div>
            <div class="anime-info">
                <div class="anime-title">${title}</div>
                <div class="anime-meta">
                    <div class="anime-date">${episodes[0]?.pubDate ? new Date(episodes[0].pubDate).toLocaleDateString() : 'N/A'}</div>
                </div>
            </div>`;
        card.addEventListener('click', () => {
            window.location.href = `detail.html?title=${encodeURIComponent(title)}`;
        });
        animeList.appendChild(card);
    });
    loadingIndicator.style.display = 'none';
}



loadAnimeData().then(data => {
    if (Object.keys(data).length) renderAnimeList(data);
});
