// app.js
App({
  // API基础URL
  apiBase: 'http://128.199.108.48:3000/api',

  globalData: {
    userInfo: null,
    openId: null,
    sessionKey: null,
    logs: [],
    // 可扩展：全局缓存、配置等
  },

  onLaunch() {
    // 本地日志
    const logs = wx.getStorageSync('logs') || [];
    logs.unshift(Date.now());
    wx.setStorageSync('logs', logs);
    this.globalData.logs = logs;

    // 登录并获取openId
    wx.login({
      success: res => {
        if (res.code) {
          // 可在此处请求后端获取openId
          // wx.request({
          //   url: `${this.apiBase}/login`,
          //   data: { code: res.code },
          //   success: r => { this.globalData.openId = r.data.openId; }
          // })
        }
      }
    });
  },

  // 全局获取用户信息
  getUserInfo(cb) {
    if (this.globalData.userInfo) {
      typeof cb === 'function' && cb(this.globalData.userInfo);
    } else {
      wx.getUserProfile ? wx.getUserProfile({
        desc: '用于完善资料',
        success: res => {
          this.globalData.userInfo = res.userInfo;
          typeof cb === 'function' && cb(res.userInfo);
        },
        fail: () => {
          wx.showToast({ title: '获取用户信息失败', icon: 'none' });
        }
      }) : wx.getUserInfo({
        success: res => {
          this.globalData.userInfo = res.userInfo;
          typeof cb === 'function' && cb(res.userInfo);
        },
        fail: () => {
          wx.showToast({ title: '获取用户信息失败', icon: 'none' });
        }
      });
    }
  },

  // 全局请求方法（简易封装）
  request(options) {
    wx.request({
      url: options.url.startsWith('http') ? options.url : `${this.apiBase}${options.url}`,
      method: options.method || 'GET',
      data: options.data || {},
      header: options.header || {},
      success: options.success,
      fail: options.fail || function () {
        wx.showToast({ title: '网络错误', icon: 'none' });
      },
      complete: options.complete
    });
  },

  // 全局错误处理
  handleError(msg) {
    wx.showToast({ title: msg || '发生错误', icon: 'none' });
  }
})
