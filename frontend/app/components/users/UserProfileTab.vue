<template>
  <div v-if="user" class="user-profile-tab">
    <section class="section">
      <h4 class="section-title">Identificação & Contato</h4>
      <div class="profile-grid">
        <UserField label="Id do Usuário" :value="user.external_id" mono readonly />
        <UserField
          label="ID WhatsApp"
          v-model="form.id_wpp"
          :editing="editing"
          :original="user.id_wpp"
          mono
        />
        <UserField
          label="Pixel de Origem"
          v-model="form.pixel_name"
          :editing="editing"
          :original="user.pixel_name"
        />
        <UserField
          label="Primeiro Nome"
          v-model="form.first_name"
          :editing="editing"
          :original="user.first_name"
        />
        <UserField
          label="Sobrenome"
          v-model="form.last_name"
          :editing="editing"
          :original="user.last_name"
        />
        <UserField
          label="E-mail"
          v-model="form.email"
          :editing="editing"
          :original="user.email"
          mono
        />
        <UserField
          label="Telefone"
          v-model="form.phone"
          :editing="editing"
          :original="user.phone"
          mono
        />
      </div>
    </section>

    <section class="section">
      <h4 class="section-title">Geolocalização & Conexão</h4>
      <div class="profile-grid">
        <UserField
          label="Cidade"
          v-model="form.city"
          :editing="editing"
          :original="user.city"
        />
        <UserField
          label="Estado (UF)"
          v-model="form.state"
          :editing="editing"
          :original="user.state"
        />
        <UserField
          label="País"
          v-model="form.country"
          :editing="editing"
          :original="user.country"
        />
        <UserField
          label="CEP"
          v-model="form.postal_code"
          :editing="editing"
          :original="user.postal_code"
        />
        <UserField
          label="IP de Acesso"
          v-model="form.client_ip"
          :editing="editing"
          :original="user.client_ip"
          mono
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import UserField from '~/components/users/UserField.vue'
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'

interface Props {
  user: TrackedUser | null
  form: Required<TrackedUserUpdateInput>
  editing: boolean
}

defineProps<Props>()
</script>

<style scoped>
.user-profile-tab {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.section-title {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0;
}

.profile-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem 1rem;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 1rem;
}
</style>
