
# Feed Uni Web

基于 Vite + Vue3 的智能喂养 Web 应用

---

## 项目简介

Feed Uni Web 是一个将原有微信小程序迁移为现代 Web 前端的项目，采用 Vue 3 + Vite 技术栈，支持智能喂食、历史记录、实时影像、个性化推荐等功能，界面风格现代，适配多端。

---

## 技术栈
- [Vue 3](https://vuejs.org/)  (Composition API, `<script setup>`)
- [Vite](https://vitejs.dev/)  (极速开发与构建工具)
- [Vue Router](https://router.vuejs.org/)  (页面路由)
- [flv.js](https://github.com/bilibili/flv.js)  (实时视频流播放)

---

## 目录结构

```
feed_vue/
├── public/                # 公共资源
├── src/
│   ├── assets/            # 静态资源（图片、图标等）
│   ├── components/        # 可复用组件
│   ├── icons/             # SVG 图标
│   ├── pages/             # 页面目录
│   │   ├── index/         # 首页
│   │   ├── feed/          # 智能喂食
│   │   ├── history/       # 历史记录
│   │   ├── camera/        # 实时影像
│   │   ├── article/       # 文章推荐
│   │   └── profile/       # 个人中心
│   ├── router.js          # 路由配置
│   ├── style.css          # 全局样式
│   └── main.js            # 入口文件
├── index.html             # 入口 HTML
├── package.json           # 项目依赖与脚本
├── vite.config.js         # Vite 配置
└── README.md              # 项目说明
```

---

## 安装与启动

1. 安装依赖

```bash
npm install
```

2. 启动开发环境

```bash
npm run dev
```

3. 构建生产环境

```bash
npm run build
```

4. 预览生产环境

```bash
npm run preview
```

---

## 主要功能

- **智能喂食**：定时定量，科学喂养
- **历史记录**：查看喂食与操作历史
- **实时影像**：远程查看实时画面
- **个性化推荐**：推送相关知识与文章
- **多端适配**：响应式设计，适配 PC 和移动端

---

## 迁移说明
- 每个小程序页面迁移为 Vue 单文件组件（.vue）
- 小程序 JS/WXML/WXSS 分别对应 script/template/style
- 公共方法与工具函数可放在 `src/utils/`
- 静态资源统一放在 `src/assets/`
- 原有样式已适配为 CSS，支持动态自适应

---

## 接口与代理
- 所有 `/api` 请求已通过 Vite 代理到 `https://feed.greenhats.me/api`
- 详见 `vite.config.js`

---

## 相关链接
- [Vue 官方文档](https://vuejs.org/)
- [Vite 官方文档](https://vitejs.dev/)
- [flv.js 文档](https://github.com/bilibili/flv.js)

---

> 本项目由 @zhangwenxing 维护。
