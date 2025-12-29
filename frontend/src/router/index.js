import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/user/cards'
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
    path: '/merchant',
    name: 'MerchantDashboard',
    component: () => import('../views/merchant/Dashboard.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
