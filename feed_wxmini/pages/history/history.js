Page({
    data: {
        feedings: [],
        loading: false,
        error: '',
        statistics: {
            today_count: 0,
            avg_score: '-'
        }
    },
    onLoad() {
        this.fetchFeedings();
        this.fetchStatistics();
    },
    onPullDownRefresh() {
        this.fetchFeedings(() => {
            this.fetchStatistics(() => wx.stopPullDownRefresh());
        });
    },
    fetchFeedings(callback) {
        this.setData({ loading: true, error: '' });
        getApp().request({
            url: 'https://feed.greenhats.me/api/feedings',
            method: 'GET',
            success: (res) => {
                if (res.statusCode === 200) {
                    this.setData({ feedings: res.data });
                } else {
                    this.setData({ error: res.data.error || '获取数据失败' });
                    getApp().handleError(res.data.error || '获取数据失败');
                }
            },
            fail: () => {
                this.setData({
                    feedings: [],
                    error: '网络错误，无法获取数据'
                });
                getApp().handleError('网络错误，无法获取数据');
            },
            complete: () => {
                this.setData({ loading: false });
                if (callback) callback();
            }
        });
    },
    fetchStatistics(callback) {
        getApp().request({
            url: 'https://feed.greenhats.me/api/stats',
            method: 'GET',
            success: (res) => {
                if (res.statusCode === 200 && res.data) {
                    this.setData({
                        statistics: {
                            today_count: res.data.today_count || 0,
                            avg_score: res.data.avg_rating !== undefined ? res.data.avg_rating : '-'
                        }
                    });
                }
            },
            fail: () => {
                this.setData({
                    statistics: {
                        today_count: 0,
                        avg_score: '-'
                    }
                });
                getApp().handleError('统计数据获取失败');
            },
            complete: () => {
                if (callback) callback();
            }
        });
    }
})