<script setup>
import { computed, ref } from 'vue'
import MatrixTable from './components/MatrixTable.vue'

const defaultGoApiUrl = import.meta.env.VITE_GO_API_URL || 'http://localhost:8080'

const goApiUrl = ref(defaultGoApiUrl)
const apiKey = ref('')
const token = ref('')

const matrixJson = ref(JSON.stringify({ matrix: [[1, 2], [3, 4], [5, 6]] }, null, 2))

const loadingToken = ref(false)
const loadingQr = ref(false)
const error = ref('')
const result = ref(null)

const loading = computed(() => loadingToken.value || loadingQr.value)

const parsedMatrixPayload = computed(() => {
  try {
    return JSON.parse(matrixJson.value)
  } catch {
    return null
  }
})

function pretty(value) {
  return JSON.stringify(value, null, 2)
}

async function getToken() {
  error.value = ''
  result.value = null
  loadingToken.value = true

  try {
    const res = await fetch(`${goApiUrl.value}/token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': apiKey.value,
      },
      body: JSON.stringify({ sub: 'frontend' }),
    })

    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data?.error || `HTTP ${res.status}`)

    token.value = data.token
  } catch (e) {
    token.value = ''
    error.value = e?.message || 'Error al obtener token'
  } finally {
    loadingToken.value = false
  }
}

async function runQr() {
  error.value = ''
  result.value = null
  loadingQr.value = true

  try {
    if (!token.value) throw new Error('Primero obtiene un token')
    if (!parsedMatrixPayload.value) throw new Error('JSON de matriz inválido')

    const res = await fetch(`${goApiUrl.value}/qr`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token.value}`,
      },
      body: JSON.stringify(parsedMatrixPayload.value),
    })

    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data?.error || `HTTP ${res.status}`)

    result.value = data
  } catch (e) {
    error.value = e?.message || 'Error al procesar QR'
  } finally {
    loadingQr.value = false
  }
}
</script>

<template>
  <main class="app">
    <header class="header">
      <h1>Matrices — QR + Stats</h1>
      <p class="sub">Calculo de matrices QR y Stats</p>
    </header>

    <section class="card">
      <div class="grid-2">
        <div>
          <label>GO API URL</label>
          <input v-model="goApiUrl" placeholder="http://localhost:8080" />
        </div>
        <div>
          <label>API_KEY (para /token)</label>
          <input v-model="apiKey" placeholder="API_KEY" />
        </div>
      </div>

      <label>Matriz de entrada (JSON)</label>
      <textarea v-model="matrixJson" spellcheck="false" />

      <div class="actions">
        <button type="button" class="btn" :disabled="loading" @click="getToken">
          {{ loadingToken ? 'Obteniendo...' : 'Obtener token' }}
        </button>
        <button type="button" class="btn primary" :disabled="loading || !token" @click="runQr">
          {{ loadingQr ? 'Procesando...' : 'Procesar QR' }}
        </button>
      </div>

      <p v-if="token" class="token"><strong>Token:</strong> {{ token }}</p>
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="!parsedMatrixPayload" class="warning">JSON inválido</p>
    </section>

    <section v-if="result" class="grid-2">
      <div class="card">
        <h2>Stats</h2>
        <pre>{{ pretty(result.stats) }}</pre>
      </div>
      <div class="card">
        <h2>Meta</h2>
        <pre>{{ pretty(result.meta) }}</pre>
      </div>
      <div class="card">
        <h2>Q</h2>
        <MatrixTable :matrix="result.q" />
      </div>
      <div class="card">
        <h2>R</h2>
        <MatrixTable :matrix="result.r" />
      </div>
    </section>
  </main>
</template>
