import { createRouter, createWebHistory } from 'vue-router'

const appTarget = import.meta.env.VITE_APP_TARGET || 'user'
const isMerchantApp = appTarget === 'merchant'

const userRoutes = [
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
    path: '/user/bind-phone',
    name: 'UserBindPhone',
    component: () => import('../views/user/BindPhone.vue')
  },
  {
    path: '/user/code',
    name: 'UserCode',
    component: () => import('../views/user/UserCode.vue')
  },
  {
    path: '/user/nickname',
    name: 'UserNickname',
    component: () => import('../views/user/Nickname.vue')
  },
  {
    path: '/user/scan-pay',
    name: 'UserScanPay',
    component: () => import('../views/user/ScanShopPay.vue')
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

const merchantRoutes = [
  {
    path: '/',
    redirect: '/merchant/login'
  },
  {
    path: '/merchant/login',
    name: 'MerchantLogin',
    component: () => import('../views/merchant/Login.vue')
  },
  {
    path: '/shop/:slug/login',
    name: 'TechnicianLogin',
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
    path: '/merchant/bind-phone',
    name: 'MerchantBindPhone',
    component: () => import('../views/merchant/BindPhone.vue')
  },
  {
    path: '/merchant/services',
    name: 'MerchantServices',
    component: () => import('../views/merchant/Services.vue')
  },
  {
    path: '/merchant/merchant-info',
    name: 'MerchantInfo',
    component: () => import('../views/merchant/MerchantInfo.vue')
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
    path: '/merchant/scan-card',
    name: 'MerchantScanCard',
    component: () => import('../views/merchant/ScanCard.vue')
  },
  {
    path: '/merchant/cards/:id',
    name: 'MerchantCardDetail',
    component: () => import('../views/merchant/CardDetail.vue')
  },
  {
    path: '/merchant/shop-manage',
    name: 'MerchantShopManage',
    component: () => import('../views/merchant/ShopManage.vue')
  },
  {
    path: '/merchant/customer-service',
    name: 'MerchantCustomerService',
    component: () => import('../views/merchant/CustomerService.vue')
  },
  {
    path: '/merchant/role-permissions/:roleKey',
    name: 'MerchantRolePermissionAdjust',
    component: () => import('../views/merchant/RolePermissionAdjust.vue')
  },
  {
    path: '/platform-admin/login',
    name: 'PlatformAdminLogin',
    component: () => import('../views/platformAdmin/Login.vue')
  },
  {
    path: '/platform-admin',
    name: 'PlatformAdminDashboard',
    component: () => import('../views/platformAdmin/Dashboard.vue')
  }
]

const routes = isMerchantApp ? merchantRoutes : userRoutes

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
