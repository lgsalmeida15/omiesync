<template>
  <div style="padding:24px;max-width:560px">
    <div class="section-title">PERFIL</div>
    <div class="card" style="padding:24px;margin-bottom:16px">
      <div style="display:flex;align-items:center;gap:16px;margin-bottom:20px">
        <div class="avatar-lg">{{ initials }}</div>
        <div>
          <p style="font-size:16px;font-weight:700">{{ auth.user?.nome }}</p>
          <p style="font-family:var(--mono);font-size:10px;color:var(--text3);margin-top:3px">{{ auth.user?.email }}</p>
          <span class="pill" style="margin-top:6px">{{ auth.user?.role }}</span>
        </div>
      </div>
      <div style="border-top:1px solid var(--border);padding-top:16px;display:flex;flex-direction:column;gap:10px">
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span style="font-family:var(--mono);font-size:11px;color:var(--text3)">TEMA</span>
          <button @click="ui.toggleTheme()" class="btn-ghost" style="font-size:12px">
            {{ ui.theme==="dark"?"Mudar para Claro":"Mudar para Escuro" }}
          </button>
        </div>
      </div>
    </div>

    <div class="card" style="padding:24px;margin-bottom:16px">
      <p style="font-weight:700;font-size:14px;margin-bottom:16px">Alterar Senha</p>
      <div style="display:flex;flex-direction:column;gap:14px">
        <div class="field"><label>NOVA SENHA</label><input v-model="pw.p1" type="password" class="input-el" placeholder="Minimo 8 caracteres" /></div>
        <div class="field"><label>CONFIRMAR SENHA</label><input v-model="pw.p2" type="password" class="input-el" placeholder="Repita a senha" /></div>
        <p v-if="pwErr" style="font-family:var(--mono);font-size:10px;color:var(--red)">{{ pwErr }}</p>
        <p v-if="pwOk" style="font-family:var(--mono);font-size:10px;color:var(--green)">Senha alterada com sucesso.</p>
        <button class="btn-primary" :disabled="savingPw" @click="savePwd" style="align-self:flex-start">{{ savingPw?"...":"Salvar Senha" }}</button>
      </div>
    </div>

    <button class="btn-danger" style="width:100%" @click="confirmLogout=true">Sair da conta</button>

    <div v-if="confirmLogout" class="modal-overlay" @mousedown.self="confirmLogout=false">
      <div class="modal-box" style="max-width:380px">
        <div class="modal-header"><span style="font-size:15px;font-weight:700">Confirmar saida</span><button @click="confirmLogout=false" class="btn-close">x</button></div>
        <div class="modal-body"><p style="font-size:13px;color:var(--text2)">Deseja encerrar sua sessao?</p></div>
        <div class="modal-footer"><button class="btn-ghost" @click="confirmLogout=false">Cancelar</button><button class="btn-danger" @click="logout">Sair</button></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { useRouter } from "vue-router"
import { useAuthStore } from "@/stores/auth"
import { useUiStore } from "@/stores/ui"
import api from "@/api/client"
const auth = useAuthStore(); const ui = useUiStore(); const router = useRouter()
const initials = computed(() => (auth.user?.nome ?? "").split(" ").map((w:string)=>w[0]).slice(0,2).join("").toUpperCase())
const pw = ref({ p1:"", p2:"" }); const pwErr=ref(""); const pwOk=ref(false); const savingPw=ref(false)
const confirmLogout = ref(false)
async function savePwd() {
  pwErr.value=""; pwOk.value=false
  if(pw.value.p1.length<8){pwErr.value="Minimo 8 caracteres";return}
  if(pw.value.p1!==pw.value.p2){pwErr.value="Senhas nao conferem";return}
  savingPw.value=true
  try {
    const grupoId = auth.user?.grupo_id
    await api.put(`/admin/grupos/${grupoId}/usuarios/${auth.user?.id}/password`,{password:pw.value.p1})
    pw.value={p1:"",p2:""}; pwOk.value=true
  } catch(e:any){pwErr.value=e?.response?.data?.message??"Erro"} finally{savingPw.value=false}
}
async function logout() {
  try { await auth.logout() } catch{}
  router.push("/login")
}
</script>

<style scoped>
.card{background:var(--card);border:1px solid var(--border);border-radius:12px}
.avatar-lg{width:52px;height:52px;border-radius:12px;background:linear-gradient(135deg,var(--accent),var(--accent3));display:flex;align-items:center;justify-content:center;font-size:18px;font-weight:800;color:#080c12;flex-shrink:0}
.pill{display:inline-flex;padding:2px 9px;border-radius:20px;font-family:var(--mono);font-size:10px;font-weight:600;background:rgba(0,229,255,0.1);color:var(--accent)}
.field{display:flex;flex-direction:column;gap:6px}
label{font-family:var(--mono);font-size:10px;color:var(--text3);letter-spacing:1.5px;text-transform:uppercase}
.input-el{background:var(--bg3);border:1px solid var(--border2);border-radius:8px;padding:9px 12px;font-size:13px;color:var(--text);outline:none;transition:border-color 0.2s}
.input-el:focus{border-color:rgba(0,229,255,0.5)}
.btn-primary{background:var(--accent);color:#080c12;border:none;border-radius:8px;padding:9px 18px;font-size:13px;font-weight:600;cursor:pointer}
.btn-primary:hover:not(:disabled){background:rgba(0,229,255,0.85)}.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
.btn-danger{background:rgba(239,68,68,0.1);color:#ef4444;border:1px solid rgba(239,68,68,0.3);border-radius:8px;padding:9px 16px;font-size:13px;font-weight:600;cursor:pointer}
.btn-ghost{background:var(--bg3);color:var(--text2);border:1px solid var(--border2);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:600;cursor:pointer}
.btn-close{background:var(--bg3);color:var(--text3);border:1px solid var(--border2);border-radius:6px;width:28px;height:28px;cursor:pointer;font-size:14px;display:flex;align-items:center;justify-content:center}
.modal-overlay{position:fixed;inset:0;z-index:1000;background:var(--overlay);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;padding:24px}
.modal-box{background:var(--card);border:1px solid var(--border2);border-radius:12px;box-shadow:var(--shadow);width:100%;display:flex;flex-direction:column}
.modal-header{display:flex;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--border)}
.modal-body{padding:24px}
.modal-footer{padding:16px 24px;border-top:1px solid var(--border);display:flex;justify-content:flex-end;gap:8px}
</style>
