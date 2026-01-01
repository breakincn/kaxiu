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
    path: '/user/register',
    name: 'UserRegister',
    component: () => import('../views/user/Register.vue')
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
    path: '/user/scan-pay',
    name: 'UserScanPay',
    component: () => import('../views/user/ScanShopPay.vue')
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
  },
  {
    path: '/merchant/issue-card',
    name: 'MerchantIssueCard',
    component: () => import('../views/merchant/IssueCard.vue')
  },
  {
    path: '/merchant/scan-verify',
    name: 'MerchantScanVerify',
    component: () => import('../views/merchant/ScanVerify.vue')
  },
  {
    path: '/merchant/shop-manage',
    name: 'MerchantShopManage',
    component: () => import('../views/merchant/ShopManage.vue')
  },
  // Shop 模块：用户扫码售卡页面
  {
    path: '/shop/:slug',
    name: 'Shop',
    component: () => import('../views/user/Shop.vue')
  },
  {
    path: '/shop/id/:id',
    name: 'ShopById',
    component: () => import('../views/user/Shop.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
