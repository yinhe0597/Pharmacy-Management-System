<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="用药咨询" name="consultations">
        <PharmaPanel
          :api="listConsultations"
          :create="createConsultation"
          :update="updateConsultation"
          :remove="deleteConsultation"
          :columns="consultColumns"
          :fields="consultFields"
        />
      </el-tab-pane>
      <el-tab-pane label="不良反应登记" name="adverse">
        <PharmaPanel
          :api="listAdverseReactions"
          :create="createAdverseReaction"
          :update="updateAdverseReaction"
          :remove="deleteAdverseReaction"
          :columns="adverseColumns"
          :fields="adverseFields"
        />
      </el-tab-pane>
      <el-tab-pane label="用药指导" name="guidances">
        <PharmaPanel
          :api="listGuidances"
          :create="createGuidance"
          :update="updateGuidance"
          :remove="deleteGuidance"
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
  updateConsultation,
  deleteConsultation,
  listAdverseReactions,
  createAdverseReaction,
  updateAdverseReaction,
  deleteAdverseReaction,
  listGuidances,
  createGuidance,
  updateGuidance,
  deleteGuidance,
} from '@/api/pharma'

const tab = ref('consultations')

const consultColumns = ['patient_name', 'question', 'answer', 'created_at']
const consultFields = ['patient_name', 'question', 'answer']

// drug_name 由后端 JOIN 药品表返回（只读展示列）；表单字段用后端真实入参名
// （drug_id / reaction_desc），此前用 drug_name/reaction 提交会被后端丢弃。
const adverseColumns = ['patient_name', 'drug_name', 'reaction_desc', 'severity', 'created_at']
const adverseFields = ['patient_name', 'drug_id', 'reaction_desc']

const guidanceColumns = ['drug_name', 'content', 'created_at']
const guidanceFields = ['patient_id', 'drug_id', 'content']
</script>
