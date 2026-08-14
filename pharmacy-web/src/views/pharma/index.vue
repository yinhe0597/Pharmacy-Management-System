<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="用药咨询" name="consultations">
        <PharmaPanel
          :api="listConsultations"
          :create="createConsultation"
          :columns="consultColumns"
          :fields="consultFields"
        />
      </el-tab-pane>
      <el-tab-pane label="不良反应登记" name="adverse">
        <PharmaPanel
          :api="listAdverseReactions"
          :create="createAdverseReaction"
          :columns="adverseColumns"
          :fields="adverseFields"
        />
      </el-tab-pane>
      <el-tab-pane label="用药指导" name="guidances">
        <PharmaPanel
          :api="listGuidances"
          :create="createGuidance"
          :columns="guidanceColumns"
          :fields="guidanceFields"
        />
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import PharmaPanel from './PharmaPanel.vue'
import {
  listConsultations,
  createConsultation,
  listAdverseReactions,
  createAdverseReaction,
  listGuidances,
  createGuidance,
} from '@/api/pharma'

const tab = ref('consultations')

const consultColumns = ['patient_name', 'question', 'answer', 'created_at']
const consultFields = ['patient_name', 'question', 'answer']

const adverseColumns = ['patient_name', 'drug_name', 'reaction', 'severity', 'created_at']
const adverseFields = ['patient_name', 'drug_name', 'reaction']

const guidanceColumns = ['drug_name', 'content', 'created_at']
const guidanceFields = ['drug_name', 'content']
</script>
