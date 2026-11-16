<script setup lang="ts">
defineProps<{
  nodes: GalgameOfficialFamilyNode[]
  currentId: number
}>()
</script>

<template>
  <ul class="space-y-2">
    <li v-for="item in nodes" :key="item.official.id" class="min-w-0 space-y-2">
      <GalgameOfficialRelationNode
        :official="item.official"
        :role="item.role"
        :is-current="item.official.id === currentId"
      />
      <GalgameOfficialRelationTree
        v-if="item.children.length"
        :nodes="item.children"
        :current-id="currentId"
        class="border-default-200 ml-3 border-l pl-2"
      />
    </li>
  </ul>
</template>
