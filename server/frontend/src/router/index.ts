import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import OverviewView from '../views/OverviewView.vue'
import SoftwareView from '../views/SoftwareView.vue'
import ClientsView from '../views/ClientsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    {
      path: '/',
      component: DashboardView,
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'overview', component: OverviewView },
        { path: 'software', name: 'software', component: SoftwareView },
        { path: 'clients', name: 'clients', component: ClientsView },
      ],
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) {
    next('/login')
  } else {
    next()
  }
})

export default router
