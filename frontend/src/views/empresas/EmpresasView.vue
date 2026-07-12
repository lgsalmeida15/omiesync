<template>
  <div style="padding:24px">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
      <div class="section-title" style="margin:0">EMPRESAS</div>
      <button v-if="!auth.isAdminGlobal" class="btn-primary" @click="openCreate">+ Nova Empresa</button>
    </div>

    <div class="table-card">
      
    <!-- Seletor de grupo para admin_global -->
    <div v-if="auth.isAdminGlobal && grupos.length > 0" style="margin-bottom:16px">
      <label style="font-family:var(--mono);font-size:10px;color:var(--text3);letter-spacing:1.5px;text-transform:uppercase;display:block;margin-bottom:6px">GRUPO</label>
      <select v-model="grupoId" class="input-el" style="max-width:320px" @change="load">
        <option value="">Selecione um grupo...</option>
        <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nome }}</option>
      </select>
    </div>
<div v-if="loading" style="padding:48px;text-align:center"><div class="spinner"></div></div>
      <div v-else-if="error" style="padding:32px;text-align:center;font-family:var(--mono);font-size:11px;color:var(--red)">{{ error }}</div>
      <div v-else-if="empresas.length===0" style="padding:48px;text-align:center;font-family:var(--mono);font-size:11px;color:var(--text3)">Nenhuma empresa cadastrada.</div>
      <div v-else style="overflow-x:auto">
        <table>
          <thead><tr>
            <th>NOME</th><th>CNPJ</th><th>APP KEY</th><th>APP SECRET</th><th>STATUS</th><th>SYNC</th><th style="text-align:right">ACOES</th>
          </tr></thead>
          <tbody>
            <tr v-for="e in empresas" :key="e.id">
              <td style="font-weight:600">{{ e.nome }}</td>
              <td style="font-family:var(--mono);font-size:11px;color:var(--text3)">{{ e.cnpj||"-" }}</td>
              <td style="font-family:var(--mono);font-size:11px;color:var(--text3)">{{ e.app_key }}</td>
              <td style="font-family:var(--mono);font-size:11px;color:var(--text3)">{{ e.app_secret }}</td>
              <td><span :class="['pill', statusCls(e.status)]">{{ e.status }}</span></td>
              <td><span :class="['pill', syncCls(e.status_sync)]">{{ e.status_sync }}</span></td>
              <td style="text-align:right;white-space:nowrap">
                <RouterLink :to="`/sync`" class="btn-ghost" style="margin-right:6px;font-size:11px">Sync</RouterLink>
                <button class="btn-ghost" @click="openEdit(e)" style="margin-right:6px">Editar</button>
                <button v-if="e.status==='deletando'" class="btn-primary" style="font-size:11px" @click="reativar(e)">Reativar</button>
                <button v-else class="btn-danger" @click="confirmDelete(e)">Excluir</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal criar/editar -->
    <div v-if="showModal" class="modal-overlay" @mousedown.self="showModal=false">
      <div class="modal-box">
        <div class="modal-header">
          <span style="font-size:15px;font-weight:700">{{ editing ? "Editar Empresa" : "Nova Empresa" }}</span>
          <button @click="showModal=false" class="btn-close">x</button>
        </div>
        <div class="modal-body">
          <div class="field"><label>NOME</label><input v-model="form.nome" class="input-el" placeholder="Razao social" /><p v-if="fe.nome" class="err">{{ fe.nome }}</p></div>
          <div class="field"><label>CNPJ</label><input v-model="form.cnpj" class="input-el" placeholder="00.000.000/0001-00" /></div>
          <div class="field"><label>APP KEY</label><input v-model="form.app_key" class="input-el" placeholder="App Key do Omie" /><p v-if="fe.app_key" class="err">{{ fe.app_key }}</p></div>
          <div class="field">
            <label>APP SECRET</label>
            <input v-model="form.app_secret" type="password" class="input-el" :placeholder="editing ? 'Deixe vazio para manter atual' : 'App Secret do Omie'" />
            <p style="font-family:var(--mono);font-size:10px;color:var(--text3);margin-top:4px">O secret nao sera exibido apos salvar.</p>
            <p v-if="fe.app_secret" class="err">{{ fe.app_secret }}</p>
          </div>
          <p v-if="saveErr" class="err-box">{{ saveErr }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="showModal=false">Cancelar</button>
          <button class="btn-primary" :disabled="saving" @click="save">{{ saving ? "..." : "Salvar" }}</button>
        </div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="showConfirm" class="modal-overlay" @mousedown.self="showConfirm=false">
      <div class="modal-box" style="max-width:420px">
        <div class="modal-header"><span style="font-size:15px;font-weight:700">Confirmar exclusao</span><button @click="showConfirm=false" class="btn-close">x</button></div>
        <div class="modal-body">
          <p style="font-size:13px;color:var(--text2)">Deseja excluir a empresa <strong style="color:var(--text)">{{ delTarget?.nome }}</strong>?</p>
          <p style="font-family:var(--mono);font-size:10px;color:var(--text3);margin-top:8px">A empresa sera marcada para exclusao em 30 dias. Voce pode reativar durante esse periodo.</p>
          <p v-if="deleteErr" class="err-box" style="margin-top:10px">{{ deleteErr }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn-ghost" @click="showConfirm=false">Cancelar</button>
          <button class="btn-danger" :disabled="deleting" @click="doDelete">{{ deleting ? "..." : "Excluir" }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import { useAuthStore } from "@/stores/auth"
import api from "@/api/client"

interface Empresa { id:string;grupo_id:string;nome:string;cnpj:string;app_key:string;app_secret:string;status:string;status_sync:string;created_at:string }

const auth = useAuthStore()
// admin_global nao tem grupo_id proprio — usa seletor
const grupos = ref<{id:string;nome:string}[]>([])
const grupoId = ref(auth.user?.grupo_id ?? "")
async function loadGrupos() {
  if(auth.isAdminGlobal.value) {
    try { const r=await api.get("/admin/grupos?page=1&per_page=100"); grupos.value=r.data.data??[] }
    catch{}
  }
}

const empresas = ref<Empresa[]>([])
const loading = ref(false)
const error = ref("")
const showModal = ref(false)
const showConfirm = ref(false)
const editing = ref<Empresa|null>(null)
const delTarget = ref<Empresa|null>(null)
const form = ref({ nome:"", cnpj:"", app_key:"", app_secret:"" })
const fe = ref({ nome:"", app_key:"", app_secret:"" })
const saveErr = ref("")
const deleteErr = ref("")
const saving = ref(false)
const deleting = ref(false)

async function load() {
  if (!grupoId.value) return
  loading.value=true; error.value=""
  try {
    const r = await api.get(`/admin/grupos/${grupoId.value}/empresas?page=1&per_page=100`)
    empresas.value = r.data.data ?? []
  } catch(e:any) { error.value = e?.response?.data?.message ?? "Erro ao carregar" }
  finally { loading.value=false }
}

function openCreate() { editing.value=null; form.value={nome:"",cnpj:"",app_key:"",app_secret:""}; fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""; showModal.value=true }
function openEdit(e:Empresa) { editing.value=e; form.value={nome:e.nome,cnpj:e.cnpj,app_key:e.app_key,app_secret:""}; fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""; showModal.value=true }
function confirmDelete(e:Empresa) { delTarget.value=e; deleteErr.value=""; showConfirm.value=true }

async function save() {
  fe.value={nome:"",app_key:"",app_secret:""}; saveErr.value=""
  if(!form.value.nome.trim()){fe.value.nome="Nome obrigatorio";return}
  if(!form.value.app_key.trim()){fe.value.app_key="App Key obrigatorio";return}
  if(!editing.value && !form.value.app_secret.trim()){fe.value.app_secret="App Secret obrigatorio";return}
  saving.value=true
  try {
    const payload:any = { nome:form.value.nome, cnpj:form.value.cnpj, app_key:form.value.app_key }
    if(form.value.app_secret) payload.app_secret = form.value.app_secret
    if(editing.value) {
      await api.put(`/admin/grupos/${grupoId.value}/empresas/${editing.value.id}`, {...payload, app_secret: form.value.app_secret||editing.value.app_secret})
    } else {
      await api.post(`/admin/grupos/${grupoId.value}/empresas`, payload)
    }
    showModal.value=false; await load()
  } catch(e:any) { saveErr.value = e?.response?.data?.message ?? "Erro ao salvar" }
  finally { saving.value=false }
}

async function doDelete() {
  if(!delTarget.value) return
  deleting.value=true; deleteErr.value=""
  try { await api.delete(`/admin/grupos/${grupoId.value}/empresas/${delTarget.value.id}`); showConfirm.value=false; await load() }
  catch(e:any) { deleteErr.value = e?.response?.data?.message ?? "Erro" }
  finally { deleting.value=false }
}

async function reativar(e:Empresa) {
  try { await api.post(`/admin/grupos/${grupoId.value}/empresas/${e.id}/reativar`); await load() }
  catch(err:any) { alert(err?.response?.data?.message ?? "Erro") }
}

function statusCls(s:string) { return s==="ativa"?"pill-green":s==="deletando"?"pill-red":"pill-gray" }
function syncCls(s:string) { return s==="ativo"?"pill-green":s==="erro"?"pill-red":s==="pausado"?"pill-yellow":"pill-gray" }

onMounted(()=>{ loadGrupos(); if(grupoId.value) load() })
</script>

<style scoped>
.table-card{background:var(--card);border:1px solid var(--border);border-radius:12px;overflow:hidden}
table{width:100%;border-collapse:collapse}
th{font-family:var(--mono);font-size:9px;letter-spacing:1.5px;text-transform:uppercase;color:var(--text3);padding:11px 18px;text-align:left;background:rgba(255,255,255,0.02);border-bottom:1px solid var(--border)}
td{padding:10px 18px;font-size:13px;color:var(--text);border-bottom:1px solid var(--border)}
tr:last-child td{border-bottom:none}tr:hover td{background:rgba(255,255,255,0.02)}
.pill{display:inline-flex;align-items:center;padding:2px 9px;border-radius:20px;font-family:var(--mono);font-size:10px;font-weight:600}
.pill-green{background:rgba(34,197,94,0.12);color:#22c55e}.pill-red{background:rgba(239,68,68,0.12);color:#ef4444}.pill-yellow{background:rgba(245,158,11,0.12);color:#f59e0b}.pill-gray{background:rgba(255,255,255,0.06);color:var(--text3)}
.btn-primary{background:var(--accent);color:#080c12;border:none;border-radius:8px;padding:8px 16px;font-size:13px;font-weight:600;cursor:pointer;transition:var(--trans);text-decoration:none;display:inline-block}
.btn-primary:hover:not(:disabled){background:rgba(0,229,255,0.85)}.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
.btn-danger{background:rgba(239,68,68,0.1);color:#ef4444;border:1px solid rgba(239,68,68,0.3);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:600;cursor:pointer;transition:var(--trans)}
.btn-danger:hover:not(:disabled){background:rgba(239,68,68,0.2)}.btn-danger:disabled{opacity:0.5;cursor:not-allowed}
.btn-ghost{background:var(--bg3);color:var(--text2);border:1px solid var(--border2);border-radius:8px;padding:6px 12px;font-size:12px;font-weight:600;cursor:pointer;transition:var(--trans);text-decoration:none;display:inline-block}
.btn-ghost:hover{border-color:var(--accent);color:var(--accent)}
.btn-close{background:var(--bg3);color:var(--text3);border:1px solid var(--border2);border-radius:6px;width:28px;height:28px;cursor:pointer;font-size:14px;display:flex;align-items:center;justify-content:center;transition:var(--trans)}
.btn-close:hover{border-color:var(--red);color:var(--red)}
.modal-overlay{position:fixed;inset:0;z-index:1000;background:var(--overlay);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;padding:24px}
.modal-box{background:var(--card);border:1px solid var(--border2);border-radius:12px;box-shadow:var(--shadow);width:100%;max-width:520px;display:flex;flex-direction:column;max-height:calc(100vh - 48px);overflow:hidden}
.modal-header{display:flex;align-items:center;justify-content:space-between;padding:18px 24px;border-bottom:1px solid var(--border);flex-shrink:0}
.modal-body{padding:24px;display:flex;flex-direction:column;gap:16px;overflow-y:auto}
.modal-footer{padding:16px 24px;border-top:1px solid var(--border);display:flex;justify-content:flex-end;gap:8px;flex-shrink:0}
.field{display:flex;flex-direction:column;gap:6px}
label{font-family:var(--mono);font-size:10px;color:var(--text3);letter-spacing:1.5px;text-transform:uppercase}
.input-el{background:var(--bg3);border:1px solid var(--border2);border-radius:8px;padding:9px 12px;font-size:13px;color:var(--text);outline:none;transition:border-color 0.2s}
.input-el:focus{border-color:rgba(0,229,255,0.5)}.input-el:disabled{opacity:0.5}
.err{font-family:var(--mono);font-size:10px;color:var(--red)}
.err-box{font-family:var(--mono);font-size:11px;color:var(--red);background:rgba(239,68,68,0.08);border:1px solid rgba(239,68,68,0.2);border-radius:7px;padding:9px 12px}
.spinner{width:24px;height:24px;border:2px solid var(--border2);border-top-color:var(--accent);border-radius:50%;animation:spin 0.7s linear infinite;margin:0 auto}
@keyframes spin{to{transform:rotate(360deg)}}
</style>


