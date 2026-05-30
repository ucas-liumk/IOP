<template>
  <Teleport to="body">
    <Transition name="imp-fade">
      <div v-if="open" class="imp-overlay" @click.self="close">
        <Transition name="imp-pop" appear>
          <div class="imp-modal" role="dialog" aria-modal="true" :aria-label="title">
            <header class="imp-head">
              <h2 class="imp-title">{{ title }}</h2>
              <button class="imp-x" type="button" aria-label="关闭" @click="close">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
                  <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </header>

            <div class="imp-body">
              <!-- Result view -->
              <div v-if="result" class="imp-result">
                <div class="imp-summary">
                  <div class="imp-stat">
                    <span class="imp-stat-num">{{ result.total }}</span>
                    <span class="imp-stat-label">总计</span>
                  </div>
                  <div class="imp-stat ok">
                    <span class="imp-stat-num">{{ result.succeeded }}</span>
                    <span class="imp-stat-label">成功</span>
                  </div>
                  <div class="imp-stat" :class="{ bad: result.failed > 0 }">
                    <span class="imp-stat-num">{{ result.failed }}</span>
                    <span class="imp-stat-label">失败</span>
                  </div>
                </div>
                <div v-if="result.errors?.length" class="imp-errors">
                  <table class="imp-err-table">
                    <thead>
                      <tr><th>行</th><th>标识</th><th>原因</th></tr>
                    </thead>
                    <tbody>
                      <tr v-for="(e, i) in result.errors" :key="i">
                        <td class="imp-err-row">{{ e.row }}</td>
                        <td class="imp-err-key">{{ e.key || '—' }}</td>
                        <td class="imp-err-msg">{{ e.message }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <p v-else class="imp-all-ok">全部导入成功。</p>
              </div>

              <!-- Picker view -->
              <div v-else>
                <p class="imp-hint">
                  上传 CSV 文件以批量导入。请先
                  <a class="imp-link" href="#" @click.prevent="downloadTemplate">下载模板</a>
                  ，按格式填写后再上传。
                </p>
                <div
                  class="imp-drop"
                  :class="{ over: dragOver, has: !!file }"
                  @dragover.prevent="dragOver = true"
                  @dragleave.prevent="dragOver = false"
                  @drop.prevent="onDrop"
                  @click="picker?.click()"
                >
                  <input
                    ref="picker"
                    type="file"
                    accept=".csv,text/csv"
                    class="imp-file-hidden"
                    @change="onPick"
                  />
                  <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" /><line x1="12" y1="3" x2="12" y2="15" />
                  </svg>
                  <div v-if="file" class="imp-file-name">{{ file.name }}</div>
                  <div v-else class="imp-drop-text">点击选择或拖拽 CSV 文件到此处</div>
                </div>
                <p v-if="errorMsg" class="imp-error-msg">{{ errorMsg }}</p>
              </div>
            </div>

            <footer class="imp-foot">
              <button v-if="result" class="btn btn-ghost" type="button" @click="reset">再次导入</button>
              <button class="btn btn-ghost" type="button" @click="close">
                {{ result ? "完成" : "取消" }}
              </button>
              <button
                v-if="!result"
                class="btn btn-primary"
                type="button"
                :disabled="!file || busy"
                @click="doImport"
              >
                {{ busy ? "导入中…" : "导入" }}
              </button>
            </footer>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { client } from "@/api/client";
import { useNotification } from "@/shell/notify";

export interface BulkRowError { row: number; key?: string; message: string }
export interface BulkResult { total: number; succeeded: number; failed: number; errors: BulkRowError[] }

const props = withDefaults(
  defineProps<{
    /** v-model:open — controls visibility. */
    open: boolean;
    title?: string;
    /** API path (relative to axios baseURL) the template CSV is fetched from. */
    templateUrl: string;
    /** API path the multipart import is POSTed to (field name "file"). */
    importUrl: string;
    /** suggested download filename for the template. */
    templateName?: string;
  }>(),
  { title: "批量导入", templateName: "template.csv" },
);

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "done", result: BulkResult): void;
}>();

const notify = useNotification();
const picker = ref<HTMLInputElement | null>(null);
const file = ref<File | null>(null);
const dragOver = ref(false);
const busy = ref(false);
const result = ref<BulkResult | null>(null);
const errorMsg = ref("");

// Reset transient state whenever the dialog (re)opens.
watch(
  () => props.open,
  (o) => {
    if (o) reset();
  },
);

function reset() {
  file.value = null;
  result.value = null;
  errorMsg.value = "";
  busy.value = false;
  dragOver.value = false;
  if (picker.value) picker.value.value = "";
}

function close() {
  emit("update:open", false);
}

function setFile(f: File | null) {
  errorMsg.value = "";
  if (f && !/\.csv$/i.test(f.name)) {
    errorMsg.value = "请选择 .csv 文件";
    return;
  }
  file.value = f;
}
function onPick(e: Event) {
  setFile((e.target as HTMLInputElement).files?.[0] ?? null);
}
function onDrop(e: DragEvent) {
  dragOver.value = false;
  setFile(e.dataTransfer?.files?.[0] ?? null);
}

async function downloadTemplate() {
  try {
    const res = await client.get(props.templateUrl, { responseType: "blob" });
    triggerBlobDownload(res.data, props.templateName);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "模板下载失败");
  }
}

async function doImport() {
  if (!file.value) return;
  busy.value = true;
  errorMsg.value = "";
  try {
    const fd = new FormData();
    fd.append("file", file.value);
    const res = await client.post(props.importUrl, fd);
    const data: BulkResult = res.data?.data ?? res.data;
    result.value = data;
    if (data.failed > 0) {
      notify.warning(`导入完成：成功 ${data.succeeded}，失败 ${data.failed}`);
    } else {
      notify.success(`导入成功：${data.succeeded} 条`);
    }
    emit("done", data);
  } catch (e: any) {
    errorMsg.value = e.response?.data?.error?.message ?? "导入失败";
    notify.error(errorMsg.value);
  } finally {
    busy.value = false;
  }
}

function triggerBlobDownload(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
</script>

<style scoped>
.imp-overlay {
  position: fixed;
  inset: 0;
  z-index: 9100;
  background: rgba(13, 27, 46, .42);
  backdrop-filter: blur(3px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.imp-modal {
  width: 100%;
  max-width: 520px;
  max-height: 86vh;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border-radius: var(--r-lg, 14px);
  box-shadow: var(--sh-4);
  overflow: hidden;
}
.imp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.imp-title { font-size: 15px; font-weight: 700; color: var(--text); }
.imp-x {
  border: 0; background: transparent; color: var(--text-3);
  width: 28px; height: 28px; border-radius: 6px;
  display: grid; place-items: center; cursor: pointer;
}
.imp-x:hover { background: var(--surface-2); color: var(--text); }
.imp-body { padding: 20px; overflow: auto; }
.imp-hint { font-size: 13px; color: var(--text-2); line-height: 1.6; margin-bottom: 14px; }
.imp-link { color: var(--primary); font-weight: 600; text-decoration: none; }
.imp-link:hover { text-decoration: underline; }
.imp-drop {
  border: 1.5px dashed var(--border);
  border-radius: var(--r-md, 10px);
  padding: 28px 16px;
  text-align: center;
  color: var(--text-3);
  cursor: pointer;
  transition: border-color .14s, background .14s, color .14s;
}
.imp-drop:hover, .imp-drop.over { border-color: var(--primary); color: var(--primary); background: var(--primary-soft); }
.imp-drop.has { border-style: solid; border-color: var(--primary); color: var(--text); }
.imp-file-hidden { display: none; }
.imp-drop-text { font-size: 13px; margin-top: 8px; }
.imp-file-name { font-size: 13px; font-weight: 600; margin-top: 8px; color: var(--text); }
.imp-error-msg { color: var(--danger); font-size: 12.5px; margin-top: 10px; }
.imp-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
}
.imp-foot .btn { cursor: pointer; }
.imp-foot .btn:disabled { opacity: .5; cursor: not-allowed; }

/* result */
.imp-summary { display: flex; gap: 10px; margin-bottom: 16px; }
.imp-stat {
  flex: 1;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r-md, 10px);
  padding: 14px;
  text-align: center;
}
.imp-stat-num { display: block; font-size: 22px; font-weight: 700; color: var(--text); font-variant-numeric: tabular-nums; }
.imp-stat-label { display: block; font-size: 12px; color: var(--text-3); margin-top: 2px; }
.imp-stat.ok .imp-stat-num { color: var(--success, #16a34a); }
.imp-stat.bad .imp-stat-num { color: var(--danger); }
.imp-all-ok { font-size: 13px; color: var(--success, #16a34a); text-align: center; padding: 8px 0; }
.imp-errors { max-height: 280px; overflow: auto; border: 1px solid var(--border); border-radius: var(--r-md, 10px); }
.imp-err-table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.imp-err-table th {
  text-align: left; font-weight: 600; color: var(--text-3);
  padding: 8px 12px; background: var(--surface-2);
  border-bottom: 1px solid var(--border);
  position: sticky; top: 0;
}
.imp-err-table td { padding: 7px 12px; border-bottom: 1px solid var(--border-soft); color: var(--text-2); }
.imp-err-table tr:last-child td { border-bottom: 0; }
.imp-err-row { width: 44px; color: var(--text-3); font-variant-numeric: tabular-nums; }
.imp-err-key { font-weight: 600; color: var(--text); white-space: nowrap; }
.imp-err-msg { color: var(--danger); }

.imp-fade-enter-active, .imp-fade-leave-active { transition: opacity .2s ease; }
.imp-fade-enter-from, .imp-fade-leave-to { opacity: 0; }
.imp-pop-enter-active { transition: transform .26s cubic-bezier(.22, 1, .36, 1), opacity .26s ease; }
.imp-pop-enter-from { transform: translateY(10px) scale(.96); opacity: 0; }
</style>
