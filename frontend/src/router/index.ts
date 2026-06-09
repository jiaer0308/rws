import { createRouter, createWebHistory } from 'vue-router'
import ReservedFund from '@/views/reserved-fund.vue'

const routes = [
  {
    path: '/',
    redirect: '/reserved-fund'
  },
  {
    path: '/reserved-fund',
    name: 'ReservedFund',
    component: ReservedFund
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
