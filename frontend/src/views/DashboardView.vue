<template>
  <div style="padding:24px">
    <div class="section-title">BEM-VINDO</div>
    <div style="background:var(--card);border:1px solid var(--border);border-radius:12px;padding:24px;margin-bottom:16px">
      <p style="font-size:16px;font-weight:700;color:var(--text)">{{ auth.user?.nome }}</p>
      <p style="font-family:var(--mono);font-size:10px;color:var(--text3);margin-top:4px;text-transform:uppercase">{{ roleLabel }}</p>
    </div>
    <div class="section-title" style="margin-top:8px">ACESSO RAPIDO</div>
    <div style="display:flex;flex-wrap:wrap;gap:10px">
      <RouterLink v-if="auth.isAdminGlobal" to="/grupos" class="quick-link">Grupos</RouterLink>
      <RouterLink v-if="auth.isAdmin" to="/empresas" class="quick-link">Empresas</RouterLink>
      <RouterLink v-if="auth.isAdmin" to="/usuarios" class="quick-link">Usuarios</RouterLink>
      <RouterLink v-if="auth.isAdmin" to="/sync" class="quick-link">Sync</RouterLink>
      <RouterLink to="/perfil" class="quick-link">Perfil</RouterLink>
    </div>
    <div style="margin-top:24px;background:var(--card);border:1px solid var(--border);border-radius:12px;padding:20px">
      <p style="font-weight:700;font-size:13px;color:var(--text)">Dashboard financeiro — Fase 2</p>
      <p style="font-family:var(--mono);font-size:11px;color:var(--text3);margin-top:6px">
        Configure uma empresa e inicie o sync para visualizar os dados financeiros.
      </p>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue"
import { useAuthStore } from "@/stores/auth"
const auth = useAuthStore()
const roleLabel = computed(() => ({
  admin_global: "Administrador Global",
  admin_grupo: "Administrador de Grupo",
  viewer: "Visualizador"
}[auth.user?.role ?? "viewer"] ?? ""))
</script>
<style scoped>
.quick-link {
  display:inline-flex;align-items:center;padding:9px 16px;border-radius:8px;
  border:1px solid var(--border2);background:var(--bg3);color:var(--text2);
  text-decoration:none;font-size:13px;font-weight:600;transition:var(--trans);
}
.quick-link:hover{border-color:var(--accent);color:var(--accent);background:rgba(0,229,255,0.06)}
</style>
