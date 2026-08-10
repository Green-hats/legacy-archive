import { createRouter, createWebHistory } from 'vue-router'

const routes = [
    { path: '/', component: () => import('./pages/index/index.vue') },
    { path: '/feed', component: () => import('./pages/feed/feed.vue') },
    { path: '/article/:id', component: () => import('./pages/article/article.vue') },
    { path: '/camera', component: () => import('./pages/camera/camera.vue') },
    { path: '/history', component: () => import('./pages/history/history.vue') },
    { path: '/profile', component: () => import('./pages/profile/profile.vue') },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

export default router
