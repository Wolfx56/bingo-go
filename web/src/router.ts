import { createRouter, createWebHistory } from 'vue-router'

// Paths are in English; page titles are user-facing and stay in French.
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/HomeView.vue'), meta: { title: 'Mes bingos' } },
    {
      path: '/new',
      component: () => import('./views/BingoFormView.vue'),
      meta: { title: 'Nouveau bingo' },
    },
    { path: '/bingos/:id(\\d+)', component: () => import('./views/BingoView.vue') },
    {
      path: '/bingos/:id(\\d+)/edit',
      component: () => import('./views/BingoFormView.vue'),
      meta: { title: 'Modifier le bingo' },
    },
    {
      path: '/history',
      component: () => import('./views/HistoryView.vue'),
      meta: { title: 'Historique' },
    },
    {
      path: '/:pathMatch(.*)*',
      component: () => import('./views/NotFoundView.vue'),
      meta: { title: 'Page introuvable' },
    },
  ],
  scrollBehavior: (_to, _from, saved) => saved ?? { top: 0 },
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  if (title) document.title = `${title} · Bingo`
})
