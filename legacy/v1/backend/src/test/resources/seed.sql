-- ================================================================
-- 种子数据 - 对齐原型 data.js (today = 2026-05-24)
-- ================================================================

INSERT INTO app_user (id, name, dept, avatar_color) VALUES
  (1,  '陈雨晴', 'CTO 办公室',  '#1e5fd9'),
  (2,  '李海明', '战略部',       '#7c4ddb'),
  (3,  '王思琪', '战略部',       '#0fa8a3'),
  (4,  '林晓帆', '客户成功部',   '#e8920e'),
  (5,  '吴楠',   '运营中心',     '#d63838'),
  (6,  '高蕾',   '人力资源部',   '#b14fa0'),
  (7,  '宋远',   '行政中心',     '#2a8856'),
  (8,  '邱明远', '行政中心',     '#1a7fb8'),
  (9,  '韩冰',   '法务合规部',   '#1e5fd9'),
  (10, '苏潇',   '品牌中心',     '#d63838'),
  (11, '罗一航', '数据中台',     '#7c4ddb'),
  (12, '潘可',   '人力资源部',   '#0fa8a3'),
  (13, '战略委员会', '协同研究', '#7c4ddb'),
  (14, '运营委员会', '运营',     '#1e5fd9'),
  (15, '运营优化办', '运营中心', '#e8920e'),
  (16, 'COO 办公室', '裁决方',   '#d63838'),
  (17, '员工代表会', '评价方',   '#2a8856');

SELECT setval(pg_get_serial_sequence('app_user','id'), 100);

-- ===== Problem #1: 研发与产品分歧 (争议路径, 会商中) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, overdue, overdue_days, latest, tags, participants) VALUES
('WT20260510-001',
 '研发与产品对 H2 路线图优先级存在分歧',
 '研发团队提出将"基础设施重构"列为 H2 P0，但产品认为应优先"客户增长功能"。两方在资源分配、风险评估、用户价值判断上分歧较大，需协同研究决策。',
 '战略规划', 'critical', 'meeting', 'dispute', 'meeting',
 1, 'CTO 办公室', '战略委员会', '公司战略部',
 '2026-05-12', '2026-05-27', 62, FALSE, 0,
 '会商已召开 2 次，主要分歧聚焦于客户流失风险评估方法的差异，预计本周内形成会商纪要。',
 '["跨部门","决策","H2 规划"]'::jsonb,
 '["产品中心","研发中心","财务部","CTO 办公室"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note, files) VALUES
('WT20260510-001','submit',  '2026-05-12 09:24', 1, '陈雨晴', 'CTO 办公室', '提报问题：H2 路线图优先级分歧，附两方初版方案。', '["H2_路线图_研发版.pdf","H2_路线图_产品版.pdf"]'::jsonb),
('WT20260510-001','review',  '2026-05-13 14:10', 2, '李海明', '战略部',    '核实问题属重大决策事项，分办至战略委员会，要求两周内闭环。', '[]'::jsonb),
('WT20260510-001','propose', '2026-05-16 16:32', 3, '王思琪', '战略部',    '研提两套折中方案，明确标记为"存在争议"，进入会商研究流程。', '["折中方案_v1.docx"]'::jsonb),
('WT20260510-001','meeting', '2026-05-21 10:15', 13, '战略委员会', '协同研究', '会商第 1 次：双方陈述意见，明确 3 处核心分歧。', '[]'::jsonb),
('WT20260510-001','meeting', '2026-05-23 15:40', 13, '战略委员会', '协同研究', '会商第 2 次：聚焦客户流失风险评估方法差异，会后将出具纪要。', '[]'::jsonb);

INSERT INTO measure (problem_id, code, title, owner, status, has_dispute, display_order) VALUES
('WT20260510-001','M1','采用基础设施先行 + 增长功能并行的混合策略','研发中心','proposed', TRUE, 1),
('WT20260510-001','M2','设立联合 OKR，每月评审节奏',                  '产品中心','proposed', FALSE, 2);

INSERT INTO dispute (id, problem_id, point, display_order) VALUES
(1, 'WT20260510-001', '基础设施 ROI 量化口径', 1),
(2, 'WT20260510-001', '客户流失风险评估',       2);

INSERT INTO dispute_position (dispute_id, party, view_text) VALUES
(1, '研发中心', '应按"未来 12 个月技术债利息"测算'),
(1, '产品中心', '应按"功能交付延迟带来的收入损失"测算'),
(2, '研发中心', '当前架构事故率 0.42%，需立即重构'),
(2, '产品中心', '事故率可接受，重构投入回报周期过长');

INSERT INTO message (problem_id, actor_user_id, actor_name, content, mentions, occurred_at) VALUES
('WT20260510-001', 1, '陈雨晴', '@战略委员会 会商纪要预计什么时候发出？', '["@战略委员会"]'::jsonb, '2026-05-22 10:05'),
('WT20260510-001', 3, '王思琪', '@陈雨晴 预计明日下班前发出，已与双方对过。', '["@陈雨晴"]'::jsonb,   '2026-05-22 11:12');

-- ===== Problem #2: 客户支持系统响应延迟 (尚未分支) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, overdue, latest, tags, participants) VALUES
('WT20260512-002',
 '客户支持系统响应延迟严重，多团队需协同优化',
 '客户反馈工单平均响应时长由 2 小时增至 6 小时。客服、客户成功、技术支持三方对瓶颈定位有分歧。',
 '运营效率', 'high', 'processing', NULL, 'propose',
 4, '客户成功部', '运营优化办', '运营中心',
 '2026-05-14', '2026-05-30', 38, FALSE,
 '已发起跨部门数据对齐，等待技术支持部提交根因分析报告。',
 '["客户体验","工单系统","协同"]'::jsonb,
 '["客户成功部","客户服务部","技术支持部"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note, files) VALUES
('WT20260512-002','submit',  '2026-05-14 11:00', 4,  '林晓帆', '客户成功部', '提交工单响应延迟问题，附 30 天工单分析。', '["工单趋势.xlsx"]'::jsonb),
('WT20260512-002','review',  '2026-05-15 09:30', 5,  '吴楠',   '运营中心',  '核实问题为高优，分办至运营优化办牵头，限期 2 周。', '[]'::jsonb),
('WT20260512-002','propose', '2026-05-21 17:20', 15, '运营优化办','运营中心','初步研提 3 项举措，正在收集数据进一步对齐。', '[]'::jsonb);

INSERT INTO measure (problem_id, code, title, owner, status, has_dispute, display_order) VALUES
('WT20260512-002','M1','增设晚高峰排班，扩容客服一线','客户服务部','proposed', FALSE, 1),
('WT20260512-002','M2','工单分级优化算法迭代',         '技术支持部','drafting', FALSE, 2),
('WT20260512-002','M3','建立跨部门 SLA 周会',           '运营优化办','proposed', FALSE, 3);

-- ===== Problem #3: 远程办公政策征求意见 (共识路径, 征求意见中) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, overdue, latest, tags, participants) VALUES
('WT20260505-003',
 '远程办公政策调整意见征集',
 'HR 拟将远程办公由每周 3 天调整为 2 天，员工反响较大，需公开征求意见。',
 '组织人事', 'normal', 'consulting', 'consensus', 'consult',
 6, '人力资源部', '人力资源部', '人力资源部',
 '2026-05-05', '2026-06-01', 70, FALSE,
 '征求意见已开启，共收集 248 条反馈，正在分类整理。',
 '["HR","政策","员工"]'::jsonb,
 '["人力资源部","员工代表会"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note, files) VALUES
('WT20260505-003','submit',  '2026-05-05 09:15', 6, '高蕾', '人力资源部', '提交政策调整草案，请求协同流程。', '["远程办公政策_草案.pdf"]'::jsonb),
('WT20260505-003','review',  '2026-05-06 14:00', 7, '宋远', '行政中心',  '核实并分办至 HR 牵头。', '[]'::jsonb),
('WT20260505-003','propose', '2026-05-12 16:30', 6, '高蕾', '人力资源部', '研提调整方案，与各业务条线对齐后无明显争议，转入征求意见。', '[]'::jsonb),
('WT20260505-003','consult', '2026-05-18 10:00', 6, '高蕾', '人力资源部', '征求意见已发布至全员，开放期 10 天。', '[]'::jsonb);

INSERT INTO measure (problem_id, code, title, owner, status, has_dispute, display_order) VALUES
('WT20260505-003','M1','远程 2 天 + 团队自定义 1 天','人力资源部','approved', FALSE, 1);

INSERT INTO consult_stat (problem_id, total_count, support_count, neutral_count, oppose_count, start_date, end_date, brief) VALUES
('WT20260505-003', 248, 142, 56, 50, '2026-05-18', '2026-05-28', '征求各业务条线对远程办公调整方案的意见，重点关注执行可行性与团队协作影响。');

-- ===== Problem #4: 南区办公室预算超支 (争议路径, 已办结) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, overdue, latest, tags, participants) VALUES
('WT20260420-004',
 '南区办公室年度装修预算超支',
 '装修实施过程发现地基隐患需额外整改，预算超支 18%，需协同财务、行政、采购三方研提应对举措。',
 '行政后勤', 'high', 'done', 'dispute', 'evaluate',
 8, '行政中心', '行政中心', '行政中心',
 '2026-04-20', '2026-05-22', 100, FALSE,
 '问题已办结，平均评分 4.6，已归档。',
 '["预算","装修","审批"]'::jsonb,
 '["行政中心","财务部","采购部"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note, files) VALUES
('WT20260420-004','submit',    '2026-04-20 10:00', 8,  '邱明远',     '行政中心', '上报预算超支情况。', '["预算超支说明.docx","现场照片.zip"]'::jsonb),
('WT20260420-004','review',    '2026-04-21 11:30', 14, '运营委员会', '运营',    '审核通过，分办至行政中心+财务部联合处理。', '[]'::jsonb),
('WT20260420-004','propose',   '2026-04-25 15:00', 8,  '邱明远',     '行政中心', '提出 3 个备选方案，财务对其中超支额度有争议。', '[]'::jsonb),
('WT20260420-004','meeting',   '2026-04-29 14:00', 14, '运营委员会', '协同',    '会商，三方对超支可接受范围达成区间共识。', '[]'::jsonb),
('WT20260420-004','arbitrate', '2026-05-04 16:00', 16, 'COO 办公室', '裁决方',  '最终裁决：批准超支 12%，剩余通过设计简化吸收。', '[]'::jsonb),
('WT20260420-004','implement', '2026-05-14 09:00', 8,  '邱明远',     '行政中心', '完成装修整改，提交完成材料。', '["完成验收单.pdf"]'::jsonb),
('WT20260420-004','evaluate',  '2026-05-21 17:00', 17, '员工代表会', '评价方',  '满意度 4.6 / 5，问题闭环。', '[]'::jsonb);

INSERT INTO measure (problem_id, code, title, owner, status, has_dispute, progress, display_order) VALUES
('WT20260420-004','M1','批准 12% 超支 + 设计简化方案','行政中心','completed', TRUE, 100, 1);

INSERT INTO dispute (id, problem_id, point, resolution, display_order) VALUES
(3, 'WT20260420-004', '可接受的超支额度', '批准 12% 超支，超出部分通过设计简化方案吸收。', 1);

INSERT INTO dispute_position (dispute_id, party, view_text) VALUES
(3, '行政中心', '需要 18% 才能完整完成'),
(3, '财务部',   '不应超过 8%，需要削减项目');

INSERT INTO evaluation (problem_id, evaluator_id, evaluator_name, party, quality, speed, collab, satisfaction, overall, comment_text, archive_best_practice) VALUES
('WT20260420-004', 17, '员工代表会', '提报方', 4.5, 4.4, 4.6, 4.5, 4.5, '裁决及时，三方协同顺畅。',                FALSE),
('WT20260420-004', 17, '员工代表会', '相关方', 4.7, 4.5, 4.8, 4.7, 4.7, '行政中心整改高效，建议后续装修预案前置评估。', TRUE);

-- ===== Problem #5: 海外合规风险 (超期) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, overdue, overdue_days, latest, tags, participants) VALUES
('WT20260418-005',
 '海外业务合规风险评估流程缺失',
 '近期海外业务扩张速度加快，合规风险评估流程不完善，多次出现回头返工。',
 '合规风控', 'critical', 'overdue', NULL, 'propose',
 9, '法务合规部', '法务合规部', '法务合规部',
 '2026-04-18', '2026-05-21', 30, TRUE, 3,
 '已超期 3 天，等待海外事业部反馈方案。',
 '["合规","风险","海外"]'::jsonb,
 '["法务合规部","海外事业部"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note) VALUES
('WT20260418-005','submit',  '2026-04-18 14:00', 9,  '韩冰',       '法务合规部', '上报合规流程缺失问题。'),
('WT20260418-005','review',  '2026-04-19 09:00', 14, '运营委员会', '运营',       '审核通过，分办至法务合规部牵头。'),
('WT20260418-005','propose', '2026-05-04 11:00', 9,  '韩冰',       '法务合规部', '初步研提方案，等待海外事业部回填本地差异。');

INSERT INTO message (problem_id, actor_user_id, actor_name, content, mentions, occurred_at) VALUES
('WT20260418-005', 9, '韩冰', '@海外事业部 请尽快反馈，已超期。', '["@海外事业部"]'::jsonb, '2026-05-22 09:00');

-- ===== Problem #6: 品牌 VI 落地 (审核中) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, latest, tags, participants) VALUES
('WT20260514-006',
 '新版品牌视觉规范全公司落地',
 '品牌中心发布新版 VI，需要协同各业务线落地。',
 '品牌市场', 'normal', 'processing', NULL, 'review',
 10, '品牌中心', '待分办', '—',
 '2026-05-16', '2026-06-07', 12,
 '审核中，待运营委员会确认承办单位。',
 '["品牌","落地","规范"]'::jsonb,
 '["品牌中心"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note, files) VALUES
('WT20260514-006','submit', '2026-05-16 11:00', 10, '苏潇', '品牌中心', '提交新版品牌落地需求。', '["VI_3.0.pdf"]'::jsonb);

-- ===== Problem #7: 数据分级标准 (刚提交) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, latest, tags, participants) VALUES
('WT20260516-007',
 '内部数据资产分级管理标准制定',
 '数据中台希望制定全公司数据资产分级标准，请协同各业务部门讨论。',
 '信息技术', 'high', 'pending', NULL, 'review',
 11, '数据中台', '—', '—',
 '2026-05-23', '2026-06-13', 5,
 '刚刚提交，等待审核。',
 '["数据","标准","协同"]'::jsonb,
 '[]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note) VALUES
('WT20260516-007','submit', '2026-05-23 16:00', 11, '罗一航', '数据中台', '提交数据分级标准建议，请求启动协同。');

-- ===== Problem #8: 员工心理健康 (共识路径, 督导落实中) =====
INSERT INTO problem (id, title, description, category, priority, status, branch, current_stage,
                     submitter_id, submitter_dept, handler_name, handler_dept,
                     submit_date, due_date, progress, latest, tags, participants) VALUES
('WT20260430-008',
 '员工心理健康支持项目立项',
 'EAP 项目立项请求，需研提资源投入、覆盖人群与频次。',
 '组织人事', 'normal', 'processing', 'consensus', 'implement',
 12, '人力资源部', '人力资源部', '人力资源部',
 '2026-04-30', '2026-06-03', 82,
 '督导落实中，已完成 80%，外包 EAP 服务商已签约。',
 '["EAP","福利","员工关怀"]'::jsonb,
 '["人力资源部","财务部","工会"]'::jsonb);

INSERT INTO stage_history (problem_id, stage, occurred_at, actor_user_id, actor_name, actor_dept, note) VALUES
('WT20260430-008','submit',    '2026-04-30 10:00', 12, '潘可',       '人力资源部', '立项申请。'),
('WT20260430-008','review',    '2026-05-01 14:00', 14, '运营委员会', '运营',       '通过审核。'),
('WT20260430-008','propose',   '2026-05-06 11:00', 12, '潘可',       '人力资源部', '研提方案，无争议。'),
('WT20260430-008','consult',   '2026-05-12 16:00', 12, '潘可',       '人力资源部', '征求意见，全员支持度 89%。'),
('WT20260430-008','implement', '2026-05-19 09:00', 12, '潘可',       '人力资源部', '督导执行，已完成签约与首批宣讲。');

INSERT INTO measure (problem_id, code, title, owner, status, has_dispute, progress, display_order) VALUES
('WT20260430-008','M1','外包 EAP + 内部督导员双轨','人力资源部','in_progress', FALSE, 80, 1);

-- 重置 dispute id 序列，避免后续插入冲突
SELECT setval(pg_get_serial_sequence('dispute','id'), 100);
SELECT setval(pg_get_serial_sequence('stage_history','id'), 1000);
SELECT setval(pg_get_serial_sequence('measure','id'), 1000);
SELECT setval(pg_get_serial_sequence('message','id'), 1000);
SELECT setval(pg_get_serial_sequence('attachment','id'), 1000);
SELECT setval(pg_get_serial_sequence('evaluation','id'), 1000);
SELECT setval(pg_get_serial_sequence('dispute_position','id'), 1000);
