<template>
  <div class="login-page">
    <div class="login-card fade-up">
      <!-- Logo -->
      <div class="login-logo">
        <div class="login-logo-icon">O</div>
        <div>
          <div class="login-logo-name">Omie<span>Sync</span></div>
          <div class="login-logo-sub">PAINEL ADMINISTRATIVO</div>
        </div>
      </div>

      <form class="login-form" @submit.prevent="submit">
        <AppInput
          v-model="email"
          label="E-mail"
          type="email"
          placeholder="seu@email.com"
          autocomplete="email"
          :error="errors.email"
        />

        <AppInput
          v-model="password"
          label="Senha"
          type="password"
          placeholder="••••••••"
          autocomplete="current-password"
          :error="errors.password"
        />

        <!-- Erro geral -->
        <p v-if="errorMsg" class="login-error">{{ errorMsg }}</p>

        <AppButton type="submit" :loading="loading" style="width:100%; margin-top:4px">
          Entrar
        </AppButton>
      </form>

      <p class="login-footer">Omie Sync · Painel Interno</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppInput  from '@/components/ui/AppInput.vue'
import AppButton from '@/components/ui/AppButton.vue'

const router   = useRouter()
const auth     = useAuthStore()
const email    = ref('')
const password = ref('')
const loading  = ref(false)
const errorMsg = ref('')
const errors   = ref({ email: '', password: '' })

async function submit() {
  errors.value  = { email: '', password: '' }
  errorMsg.value = ''

  if (!email.value)    { errors.value.email    = 'E-mail obrigatório'; return }
  if (!password.value) { errors.value.password = 'Senha obrigatória';  return }

  loading.value = true
  try {
    await auth.login(email.value, password.value)
    router.push('/')
  } catch (e: any) {
    const msg = e?.response?.data?.message
    errorMsg.value = msg === 'credenciais inválidas'
      ? 'E-mail ou senha incorretos.'
      : (msg ?? 'Erro ao conectar. Tente novamente.')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  padding: 24px;
}

.login-card {
  width: 100%; max-width: 420px;
  background: var(--card);
  border: 1px solid var(--border2);
  border-radius: var(--radius);
  padding: 40px 36px;
  box-shadow: var(--shadow);
  position: relative; z-index: 1;
}

.login-logo {
  display: flex; align-items: center; gap: 14px;
  margin-bottom: 36px;
}

.login-logo-icon {
  width: 44px; height: 44px; border-radius: 12px;
  background: linear-gradient(135deg, var(--accent), var(--accent3));
  display: flex; align-items: center; justify-content: center;
  font-size: 20px; font-weight: 800; color: #080c12; flex-shrink: 0;
  box-shadow: 0 0 24px rgba(0,229,255,0.3);
}

.login-logo-name { font-size: 22px; font-weight: 800; color: var(--text); }
.login-logo-name span { color: var(--accent); }
.login-logo-sub { font-family: var(--mono); font-size: 9px; color: var(--text3); letter-spacing: 2px; margin-top: 2px; }

.login-form { display: flex; flex-direction: column; gap: 18px; }

.login-error {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--red);
  background: rgba(239,68,68,0.08);
  border: 1px solid rgba(239,68,68,0.2);
  border-radius: 7px;
  padding: 9px 12px;
}

.login-footer {
  text-align: center;
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text3);
  margin-top: 28px;
}
</style>
