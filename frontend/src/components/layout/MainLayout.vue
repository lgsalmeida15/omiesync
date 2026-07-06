<template>
  <div class="layout">
    <AppSidebar :mobile-open="mobileOpen" @close="mobileOpen = false" />

    <main class="main-content">
      <AppTopbar @toggle-sidebar="toggleSidebar" />

      <div class="page-content">
        <RouterView v-slot="{ Component }">
          <Transition name="fade-page" mode="out-in">
            <component :is="Component" :key="$route.path" />
          </Transition>
        </RouterView>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppSidebar from './AppSidebar.vue'
import AppTopbar  from './AppTopbar.vue'

const mobileOpen = ref(false)

function toggleSidebar() {
  if (window.innerWidth <= 768) {
    mobileOpen.value = !mobileOpen.value
  }
}
</script>

<style scoped>
.layout { display: flex; min-height: 100vh; }

.main-content {
  margin-left: var(--sidebar-w);
  flex: 1; min-width: 0;
  position: relative; z-index: 1;
  transition: margin-left 0.28s cubic-bezier(0.4,0,0.2,1);
}

@media (max-width: 768px) {
  .main-content { margin-left: 0 !important; }
}

.page-content { padding: var(--space-xl); }
@media (max-width: 1023px) { .page-content { padding: var(--space-lg); } }
@media (max-width: 767px)  { .page-content { padding: var(--space-md); } }

.fade-page-enter-active, .fade-page-leave-active { transition: opacity 0.18s ease, transform 0.18s ease; }
.fade-page-enter-from { opacity: 0; transform: translateY(6px); }
.fade-page-leave-to  { opacity: 0; transform: translateY(-6px); }
</style>
