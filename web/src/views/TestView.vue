<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAdminServiceClient } from '../client'

const client = useAdminServiceClient()
const isLoaded = ref(false)
const active = ref<string[]>([])
const decisions = ref<string[]>([])
const error = ref<string | null>(null)

onMounted(async () => {
  await fetchStatus()
})

const fetchStatus = async () => {
  try {
    const response = await client.listDecisions({})
    isLoaded.value = true
    decisions.value = response.decision.map((x) => x.name)
  } catch (err) {
    error.value = (err as Error).message
  }
}
</script>

<template>
  <div class="test">
    <div v-if="isLoaded">
      <h1>Connected!</h1>

      <div>
        <h2>Active packages</h2>
        <ul v-if="active.length">
          <li v-for="item in active" :key="item">{{ item }}</li>
        </ul>
        <p v-else>No active packages currently available.</p>
      </div>

      <div>
        <h2>Awaiting packages</h2>
        <ul v-if="decisions.length">
          <li v-for="item in decisions" :key="item">{{ item }}</li>
        </ul>
        <p v-else>No decisions currently available.</p>
      </div>
    </div>

    <div v-if="!isLoaded">
      <h1>Connection error</h1>
      <div>
        {{ error }}
      </div>
    </div>

    <hr />
    <button @click="fetchStatus">Reload</button>
  </div>
</template>

<style>
@media (min-width: 1024px) {
  .test {
    min-height: 100vh;
    display: flex;
    justify-content: center; /* Center items vertically */
    flex-direction: column;
  }
  h2 {
    margin-top: 4rem;
  }
}

hr {
  margin: 30px 0;
}
</style>
