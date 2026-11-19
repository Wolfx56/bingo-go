<template>
  <div class="page">
    <v-empty-state headline="Page introuvable" text="Cette adresse ne correspond à aucune page.">
      <template #actions>
        <v-btn to="/" color="primary" variant="flat">Retour aux bingos</v-btn>
      </template>
    </v-empty-state>
  </div>
</template>
