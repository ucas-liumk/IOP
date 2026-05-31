<template>
  <section class="admin-page">
    <PageHeader title="发起审批" :sub="picked ? picked.name : '选择一个审批模板开始'">
      <template #actions>
        <button v-if="picked" class="btn btn-ghost" @click="picked = null">← 重新选择</button>
      </template>
    </PageHeader>

    <!-- Step 1: pick a form -->
    <div v-if="!picked">
      <div v-if="loading" class="ap-loading">加载模板…</div>
      <EmptyState v-else-if="forms.length === 0" title="暂无可用模板" sub="请联系管理员在「模板管理」中创建" icon="▤" />
      <div v-else class="ap-grid">
        <button v-for="f in forms" :key="f.id" class="ap-tpl card clickable" @click="pick(f)">
          <span class="ap-tpl-ico" aria-hidden="true">▤</span>
          <span class="ap-tpl-name">{{ f.name }}</span>
          <span class="ap-tpl-desc">{{ f.description || "—" }}</span>
        </button>
      </div>
    </div>

    <!-- Step 2: dynamic form -->
    <div v-else class="card card-pad ap-form">
      <div v-for="field in picked.fields" :key="field.key" class="ap-field">
        <label class="ap-label">
          {{ field.label }}
          <span v-if="field.required" class="ap-req">*</span>
        </label>

        <input v-if="field.type === 'text'" v-model="values[field.key]" class="input" :placeholder="field.label" />
        <textarea v-else-if="field.type === 'textarea'" v-model="values[field.key]" class="input ap-textarea" :placeholder="field.label"></textarea>
        <input v-else-if="field.type === 'number'" v-model="values[field.key]" type="number" class="input" :placeholder="field.label" />
        <input v-else-if="field.type === 'date'" v-model="values[field.key]" type="date" class="input" />
        <select v-else-if="field.type === 'select'" v-model="values[field.key]" class="input">
          <option value="">请选择</option>
          <option v-for="o in field.options" :key="o" :value="o">{{ o }}</option>
        </select>
        <div v-else-if="field.type === 'radio'" class="ap-radios">
          <label v-for="o in field.options" :key="o" class="ap-radio">
            <input type="radio" :value="o" v-model="values[field.key]" /> {{ o }}
          </label>
        </div>
        <input v-else v-model="values[field.key]" class="input" :placeholder="field.label" />
      </div>

      <div v-if="picked.fields.length === 0" class="ap-empty-fields">该模板没有自定义字段，可直接提交。</div>

      <!-- Flow preview -->
      <div v-if="picked.flow.length" class="ap-flow-preview">
        <span class="ap-flow-title">审批流程</span>
        <div class="ap-flow-row">
          <span class="ap-flow-me">我</span>
          <template v-for="(n, i) in picked.flow" :key="i">
            <span class="ap-flow-arrow">→</span>
            <span class="ap-flow-node" :class="{ cc: n.type === 'cc' }">{{ nodeLabel(n) }}</span>
          </template>
        </div>
      </div>

      <div class="ap-form-footer">
        <button class="btn btn-primary" :disabled="submitting" @click="doSubmit">
          {{ submitting ? "提交中…" : "提交审批" }}
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { PageHeader, EmptyState } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import * as api from "../api";
import type { Form, FlowNode } from "../api";

const router = useRouter();
const notify = useNotification();

const forms = ref<Form[]>([]);
const loading = ref(false);
const picked = ref<Form | null>(null);
const values = reactive<Record<string, string>>({});
const submitting = ref(false);

async function load() {
  loading.value = true;
  try {
    forms.value = await api.listForms(false);
  } catch {
    notify.error("加载模板失败");
  } finally {
    loading.value = false;
  }
}

function pick(f: Form) {
  picked.value = f;
  Object.keys(values).forEach((k) => delete values[k]);
  for (const field of f.fields) values[field.key] = "";
}

function nodeLabel(n: FlowNode): string {
  const who =
    n.assignee_type === "dept_leader" ? "部门负责人" : n.assignee_type === "role" ? `角色:${n.role_code}` : "指定人";
  return n.type === "cc" ? `抄送(${who})` : who;
}

async function doSubmit() {
  if (!picked.value) return;
  for (const field of picked.value.fields) {
    if (field.required && !String(values[field.key] ?? "").trim()) {
      notify.error(`请填写「${field.label}」`);
      return;
    }
  }
  submitting.value = true;
  try {
    const ins = await api.submit(picked.value.id, { ...values });
    notify.success("已提交");
    router.push(`/approval/instances/${ins.id}`);
  } catch (e: unknown) {
    notify.error(errMsg(e) || "提交失败");
  } finally {
    submitting.value = false;
  }
}

function errMsg(e: unknown): string {
  const r = (e as { response?: { data?: { error?: { message?: string } } } })?.response;
  return r?.data?.error?.message ?? "";
}

onMounted(load);
</script>

<style scoped>
.ap-loading { padding: var(--sp-6); color: var(--text-3); }
.ap-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--sp-4);
}
.ap-tpl {
  display: flex; flex-direction: column; align-items: flex-start; gap: var(--sp-2);
  padding: var(--sp-5); text-align: left; border: none; background: var(--surface);
}
.ap-tpl-ico {
  width: 40px; height: 40px; border-radius: var(--r-md);
  display: grid; place-items: center; font-size: 20px;
  background: var(--cat-workflow); color: #fff;
}
.ap-tpl-name { font-weight: 600; color: var(--text); }
.ap-tpl-desc { font-size: 12px; color: var(--text-3); }

.ap-form { max-width: 640px; }
.ap-field { margin-bottom: var(--sp-4); }
.ap-label { display: block; font-size: 13px; font-weight: 600; color: var(--text-2); margin-bottom: var(--sp-2); }
.ap-req { color: var(--danger); }
.ap-textarea { min-height: 84px; resize: vertical; }
.ap-radios { display: flex; gap: var(--sp-4); flex-wrap: wrap; }
.ap-radio { display: inline-flex; align-items: center; gap: 6px; font-size: 14px; }
.ap-empty-fields { color: var(--text-3); font-size: 14px; margin-bottom: var(--sp-4); }

.ap-flow-preview { margin: var(--sp-4) 0; padding-top: var(--sp-4); border-top: 1px solid var(--border-soft); }
.ap-flow-title { font-size: 13px; font-weight: 600; color: var(--text-2); }
.ap-flow-row { display: flex; align-items: center; gap: var(--sp-2); flex-wrap: wrap; margin-top: var(--sp-3); }
.ap-flow-me, .ap-flow-node {
  padding: 4px 10px; border-radius: var(--r-pill); font-size: 12px;
  background: var(--primary-soft); color: var(--primary);
}
.ap-flow-node.cc { background: var(--teal-soft); color: var(--teal); }
.ap-flow-arrow { color: var(--text-4); }

.ap-form-footer { margin-top: var(--sp-5); display: flex; justify-content: flex-end; }
</style>
