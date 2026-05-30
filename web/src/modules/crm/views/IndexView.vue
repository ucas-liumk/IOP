<template>
  <section>
    <PageHeader title="客户管理 CRM" sub="由脚手架生成 · 替换为真实功能">
      <template #actions>
        <button class="btn btn-primary" @click="showCreate = !showCreate">+ 新建</button>
      </template>
    </PageHeader>

    <article v-if="showCreate" class="card create-form">
      <form @submit.prevent="create">
        <div class="row">
          <input class="input" v-model="form.title" placeholder="标题" required />
          <input class="input" v-model="form.body" placeholder="内容" />
          <button class="btn btn-primary" type="submit">提交</button>
        </div>
      </form>
    </article>

    <DataTable :columns="columns" :rows="items" rowKey="id" emptyText="暂无数据">
      <template #cell-title="{ row }">
        <strong>{{ row.title }}</strong>
      </template>
      <template #cell-created_at="{ row }">
        <span class="time">{{ row.created_at?.slice(0, 10) }}</span>
      </template>
    </DataTable>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, type Column } from "@/shell/components";
import { createItem, listItems, type CrmItem } from "../api/crm";

const items = ref<CrmItem[]>([]);
const showCreate = ref(false);
const form = reactive({ title: "", body: "" });

const columns: Column[] = [
  { key: "title", label: "标题" },
  { key: "body", label: "内容" },
  { key: "created_at", label: "创建时间", width: "140px" },
];

onMounted(async () => { items.value = await listItems(); });

async function create() {
  if (!form.title) return;
  try {
    await createItem(form.title, form.body);
    form.title = ""; form.body = "";
    showCreate.value = false;
    items.value = await listItems();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "创建失败"); }
}
</script>

<style scoped>
.create-form { padding: 14px; margin-bottom: 14px; }
.row { display: grid; grid-template-columns: 1fr 2fr auto; gap: 10px; }
.input { padding: 8px 12px; border: 1px solid var(--border-strong); border-radius: 7px; font-size: 13px; }
.btn { padding: 8px 16px; border-radius: 7px; font-size: 13px; cursor: pointer; border: 1px solid var(--border); background: var(--surface); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.time { font-family: var(--ff-mono); color: var(--text-3); font-size: 12px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
</style>
