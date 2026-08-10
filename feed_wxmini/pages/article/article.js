Page({
    onLoad(options) {
        this.setData({ articleId: options.id })
    },
    data: {
        articleId: null
    }
    // 可扩展：文章详情相关逻辑
})
