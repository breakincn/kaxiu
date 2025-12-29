import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/user/login'
  },
  {
    path: '/user/login',
    name: 'UserLogin',
    component: () => import('../views/user/Login.vue')
  },
  {
    path: '/user/cards',
    name: 'UserCards',
    component: () => import('../views/user/CardList.vue')
  },
  {
    path: '/user/cards/:id',
    name: 'CardDetail',
    component: () => import('../views/user/CardDetail.vue')
  },
  {
    path: '/user/settings',
    name: 'UserSettings',
    component: () => import('../views/user/Settings.vue')
  },
  {
    path: '/merchant/login',
    name: 'MerchantLogin',
    component: () => import('../views/merchant/Login.vue')
  },
  {
    path: '/merchant',
    name: 'MerchantDashboard',
    component: () => import('../views/merchant/Dashboard.vue')
  },
  {
    path: '/merchant/settings',
    name: 'MerchantSettings',
    component: () => import('../views/merchant/Settings.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
