// index.js
const defaultAvatarUrl = 'https://mmbiz.qpic.cn/mmbiz/icTdbqWNOwNRna42FI242Lcia07jQodd2FJGIYQfG0LAJGFxM4FbnQP6yfMxBgJ0F3YRqJCJ1aPAK2dQagdusBZg/0'

Page({
  data: {
    motto: 'Hello World',
    userInfo: {
      avatarUrl: defaultAvatarUrl,
      nickName: '',
    },
    hasUserInfo: false,
    canIUseGetUserProfile: wx.canIUse('getUserProfile'),
    canIUseNicknameComp: wx.canIUse('input.type.nickname'),
    searchValue: '',
  },
  // 页面加载
  onLoad() {
    // 可在此处做初始化，如获取本地缓存信息
  },
  // 页面显示
  onShow() {
    // 可在此处做数据刷新
  },

  bindViewTap() {
    wx.navigateTo({
      url: '../logs/logs'
    })
  },
  onChooseAvatar(e) {
    const { avatarUrl } = e.detail
    const { nickName } = this.data.userInfo
    this.setData({
      "userInfo.avatarUrl": avatarUrl,
      hasUserInfo: nickName && avatarUrl && avatarUrl !== defaultAvatarUrl,
    })
  },
  onInputChange(e) {
    const nickName = e.detail.value
    const { avatarUrl } = this.data.userInfo
    this.setData({
      "userInfo.nickName": nickName,
      hasUserInfo: nickName && avatarUrl && avatarUrl !== defaultAvatarUrl,
    })
  },
  // 获取用户信息（兼容处理）
  getUserProfile(e) {
    getApp().getUserInfo((userInfo) => {
      this.setData({
        userInfo,
        hasUserInfo: true
      })
    });
  },
  // 跳转到智能喂食页面
  navigateToFeed() {
    wx.navigateTo({ url: '/pages/feed/feed' });
  },
  // 跳转到历史记录页面
  navigateToHistory() {
    wx.navigateTo({ url: '/pages/history/history' })
  },
  // 跳转到实时影像页面
  navigateToCamera() {
    wx.navigateTo({ url: '/pages/camera/camera' })
  },
  // 跳转到推荐文章详情页面
  navigateToArticle(e) {
    const id = e.currentTarget.dataset.id;
    wx.navigateTo({ url: `/pages/article/article?id=${id}` })
  },
  // 跳转到个人中心页面
  navigateToProfile() {
    wx.navigateTo({ url: '/pages/profile/profile' })
  },
})
