package com.gallant.collab.service;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.gallant.collab.common.BizException;
import com.gallant.collab.common.PageQuery;
import com.gallant.collab.common.PageResult;
import com.gallant.collab.common.UserContext;
import com.gallant.collab.domain.*;
import com.gallant.collab.dto.ProblemDtos;
import com.gallant.collab.dto.ProblemDtos.*;
import com.gallant.collab.mapper.*;
import com.gallant.collab.stage.Stage;
import com.gallant.collab.stage.StageEngine;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

/**
 * 问题主服务: CRUD + 状态机推进。
 * 所有写操作都会:
 *  1. 校验阶段合法性
 *  2. 写一条 stage_history
 *  3. 必要时更新 problem.status / current_stage / branch / latest / progress
 */
@Service
@RequiredArgsConstructor
public class ProblemService {

    private final ProblemMapper problemMapper;
    private final StageHistoryMapper historyMapper;
    private final MeasureMapper measureMapper;
    private final DisputeMapper disputeMapper;
    private final DisputePositionMapper positionMapper;
    private final MessageMapper messageMapper;
    private final AttachmentMapper attachmentMapper;
    private final ConsultStatMapper consultMapper;
    private final EvaluationMapper evaluationMapper;
    private final AppUserMapper userMapper;
    private final IdGenerator idGenerator;
    private final StageEngine stageEngine;

    // ============================ 查询 ============================

    public PageResult<Problem> page(PageQuery q, String status, String stage, String priority,
                                    String tab, String query) {
        QueryWrapper<Problem> w = new QueryWrapper<>();
        if (status != null && !status.isBlank() && !"all".equals(status)) {
            if ("overdue".equals(status)) w.eq("overdue", true);
            else w.eq("status", status);
        }
        if (stage != null && !stage.isBlank() && !"all".equals(stage)) {
            w.eq("current_stage", stage);
        }
        if (priority != null && !priority.isBlank() && !"all".equals(priority)) {
            w.eq("priority", priority);
        }
        if (query != null && !query.isBlank()) {
            w.and(qw -> qw.like("id", query).or().like("title", query).or().like("description", query));
        }
        Long uid = currentUid();
        if ("mine".equals(tab))     w.eq("submitter_id", uid);
        if ("overdue".equals(tab))  w.eq("overdue", true);
        if ("done".equals(tab))     w.eq("status", "done");
        // "assigned" 用 jsonb 包含查询当前用户所在部门
        if ("assigned".equals(tab)) {
            String dept = UserContext.get().dept();
            w.apply("participants @> {0}::jsonb", "[\"" + dept + "\"]");
        }
        w.orderByDesc("submit_date");

        Page<Problem> page = Page.of(q.getPage(), q.getSize());
        Page<Problem> result = problemMapper.selectPage(page, w);
        return new PageResult<>(result.getRecords(), result.getTotal(), q.getPage(), q.getSize());
    }

    public ProblemDetail detail(String id) {
        Problem p = required(id);
        AppUser submitter = userMapper.selectById(p.getSubmitterId());

        List<StageHistory> history = historyMapper.selectList(
                new QueryWrapper<StageHistory>().eq("problem_id", id).orderByAsc("occurred_at"));

        List<Measure> measures = measureMapper.selectList(
                new QueryWrapper<Measure>().eq("problem_id", id).orderByAsc("display_order"));

        List<Dispute> disputes = disputeMapper.selectList(
                new QueryWrapper<Dispute>().eq("problem_id", id).orderByAsc("display_order"));
        List<DisputeWithPositions> dwps = new ArrayList<>();
        for (Dispute d : disputes) {
            List<DisputePosition> ps = positionMapper.selectList(
                    new QueryWrapper<DisputePosition>().eq("dispute_id", d.getId()));
            dwps.add(new DisputeWithPositions(d, ps));
        }

        List<Message> messages = messageMapper.selectList(
                new QueryWrapper<Message>().eq("problem_id", id).orderByAsc("occurred_at"));

        List<Attachment> attachments = attachmentMapper.selectList(
                new QueryWrapper<Attachment>().eq("problem_id", id).orderByAsc("uploaded_at"));

        ConsultStat consult = consultMapper.selectById(id);

        List<Evaluation> evaluations = evaluationMapper.selectList(
                new QueryWrapper<Evaluation>().eq("problem_id", id));

        return new ProblemDetail(p, submitter, history, measures, dwps, messages, attachments, consult, evaluations);
    }

    // ============================ 创建 ============================

    @Transactional
    public Problem create(CreateProblemRequest req) {
        UserContext.CurrentUser u = UserContext.get();
        Problem p = new Problem();
        p.setId(idGenerator.nextProblemId());
        p.setTitle(req.getTitle());
        p.setDescription(req.getDescription());
        p.setCategory(req.getCategory());
        p.setPriority(req.getPriority());
        p.setStatus("pending");
        p.setBranch(null);
        p.setCurrentStage("review");
        p.setSubmitterId(u.id());
        p.setSubmitterDept(u.dept());
        p.setHandlerName("—");
        p.setHandlerDept("—");
        p.setSubmitDate(LocalDate.now());
        p.setDueDate(req.getDueDate());
        p.setProgress(5);
        p.setOverdue(false);
        p.setOverdueDays(0);
        p.setLatest("刚刚提交，等待审核。");
        p.setTags(req.getTags() != null ? req.getTags() : List.of());
        p.setParticipants(req.getParticipants() != null ? req.getParticipants() : List.of());
        problemMapper.insert(p);

        recordHistory(p.getId(), Stage.SUBMIT, "提报问题：" + req.getTitle(), req.getFileNames(), null);
        return p;
    }

    // ============================ 审核 ============================

    @Transactional
    public Problem review(String id, ReviewActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.REVIEW);

        if ("reject".equals(req.getDecision())) {
            p.setStatus("done"); // 一票否决
            p.setLatest("审核未通过: " + req.getReviewNote());
            problemMapper.updateById(p);
            recordHistory(id, Stage.REVIEW, "不予受理: " + req.getReviewNote(), null, null);
            return p;
        }
        if ("modify".equals(req.getDecision())) {
            p.setLatest("退回补充材料: " + req.getReviewNote());
            problemMapper.updateById(p);
            recordHistory(id, Stage.REVIEW, "退回补充材料: " + req.getReviewNote(), null, null);
            return p;
        }
        // approve
        if (req.getHandlerName() == null || req.getHandlerDept() == null) {
            throw new BizException("通过审核必须指定承办单位");
        }
        p.setHandlerName(req.getHandlerName());
        p.setHandlerDept(req.getHandlerDept());
        if (req.getPriority() != null) p.setPriority(req.getPriority());
        if (req.getDueDate() != null) p.setDueDate(req.getDueDate());
        p.setCurrentStage(Stage.PROPOSE.code);
        p.setStatus("processing");
        p.setProgress(15);
        p.setLatest("已分办至 " + req.getHandlerDept() + " 牵头研提举措。");
        problemMapper.updateById(p);

        recordHistory(id, Stage.REVIEW,
                "通过审核，分办至 " + req.getHandlerDept() + (req.getAssignNote() != null ? " · " + req.getAssignNote() : ""),
                null, null);
        return p;
    }

    // ============================ 研提举措 (关键分支节点) ============================

    @Transactional
    public Problem propose(String id, ProposeActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.PROPOSE);

        // 同步举措 (全量替换)
        measureMapper.delete(new QueryWrapper<Measure>().eq("problem_id", id));
        int order = 0;
        if (req.getMeasures() != null) {
            for (ProposeActionRequest.MeasureInput m : req.getMeasures()) {
                Measure mm = new Measure();
                mm.setProblemId(id);
                mm.setCode(m.getCode() != null ? m.getCode() : "M" + (order + 1));
                mm.setTitle(m.getTitle());
                mm.setOwner(m.getOwner());
                mm.setStatus("proposed");
                mm.setHasDispute(Boolean.TRUE.equals(m.getHasDispute()));
                mm.setProgress(0);
                mm.setDisplayOrder(++order);
                measureMapper.insert(mm);
            }
        }

        // 同步争议点 (仅 dispute 分支)
        disputeMapper.delete(new QueryWrapper<Dispute>().eq("problem_id", id));
        if (req.getHasDispute() && req.getDisputes() != null) {
            int dOrder = 0;
            for (ProposeActionRequest.DisputeInput d : req.getDisputes()) {
                Dispute dispute = new Dispute();
                dispute.setProblemId(id);
                dispute.setPoint(d.getPoint());
                dispute.setDisplayOrder(++dOrder);
                disputeMapper.insert(dispute);
                if (d.getPositions() != null) {
                    for (ProposeActionRequest.DisputePositionInput pos : d.getPositions()) {
                        DisputePosition dp = new DisputePosition();
                        dp.setDisputeId(dispute.getId());
                        dp.setParty(pos.getParty());
                        dp.setView(pos.getView());
                        positionMapper.insert(dp);
                    }
                }
            }
        }

        Stage.Branch branch = req.getHasDispute() ? Stage.Branch.DISPUTE : Stage.Branch.CONSENSUS;
        Stage next = stageEngine.advance(p, branch);
        p.setBranch(branch.code());
        p.setCurrentStage(next.code);
        p.setStatus(next.deriveProblemStatus(false));
        p.setProgress(38);
        p.setLatest("已完成研提，进入「" + next.label + "」"
                + (branch == Stage.Branch.DISPUTE ? "·" + req.getDisputes().size() + "处争议" : ""));
        problemMapper.updateById(p);

        recordHistory(id, Stage.PROPOSE, req.getNote(), null, branch.code());
        return p;
    }

    // ============================ 会商 ============================

    @Transactional
    public Problem meeting(String id, MeetingActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.MEETING);

        StringBuilder note = new StringBuilder("会商: ").append(req.getSummary());
        if (req.getAttendees() != null) note.append(" · 参会: ").append(req.getAttendees());
        recordHistory(id, Stage.MEETING, note.toString(), null, null);

        if (Boolean.TRUE.equals(req.getAdvance())) {
            Stage next = stageEngine.advance(p, Stage.Branch.DISPUTE);
            p.setCurrentStage(next.code);
            p.setStatus(next.deriveProblemStatus(false));
            p.setProgress(60);
            p.setLatest("会商完成，进入争议裁决。");
            problemMapper.updateById(p);
        } else {
            p.setLatest("新增 1 轮会商纪要。");
            problemMapper.updateById(p);
        }
        return p;
    }

    // ============================ 裁决 ============================

    @Transactional
    public Problem arbitrate(String id, ArbitrateActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.ARBITRATE);

        if (req.getResolutions() != null) {
            for (ArbitrateActionRequest.DisputeResolution r : req.getResolutions()) {
                Dispute d = disputeMapper.selectById(r.getDisputeId());
                if (d != null && id.equals(d.getProblemId())) {
                    d.setResolution(r.getResolution());
                    disputeMapper.updateById(d);
                }
            }
        }

        Stage next = stageEngine.advance(p, Stage.Branch.DISPUTE);
        p.setCurrentStage(next.code);
        p.setStatus(next.deriveProblemStatus(false));
        p.setProgress(72);
        p.setLatest("裁决发布: " + req.getOverall());
        problemMapper.updateById(p);

        recordHistory(id, Stage.ARBITRATE, "由 " + req.getArbitrator() + " 裁决: " + req.getOverall(), null, null);
        return p;
    }

    // ============================ 征求意见 ============================

    @Transactional
    public Problem consult(String id, ConsultActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.CONSULT);

        ConsultStat stat = consultMapper.selectById(id);
        if (stat == null) {
            stat = new ConsultStat();
            stat.setProblemId(id);
            stat.setTotalCount(0);
            stat.setSupportCount(0);
            stat.setNeutralCount(0);
            stat.setOpposeCount(0);
        }
        stat.setBrief(req.getBrief());
        stat.setStartDate(req.getStartDate());
        stat.setEndDate(req.getEndDate());
        stat.setRevision(req.getRevision());
        stat.setUpdatedAt(LocalDateTime.now());
        if (consultMapper.selectById(id) == null) consultMapper.insert(stat);
        else consultMapper.updateById(stat);

        recordHistory(id, Stage.CONSULT, req.getNote(), null, null);

        if (Boolean.TRUE.equals(req.getAdvance())) {
            Stage next = stageEngine.advance(p, Stage.Branch.CONSENSUS);
            p.setCurrentStage(next.code);
            p.setStatus(next.deriveProblemStatus(false));
            p.setProgress(72);
            p.setLatest("征求意见结束，进入督导落实。");
        } else {
            p.setLatest("征求意见进行中。");
        }
        problemMapper.updateById(p);
        return p;
    }

    // ============================ 督导 ============================

    @Transactional
    public Problem implement(String id, ImplementActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.IMPLEMENT);

        if (req.getMeasureProgress() != null) {
            for (ImplementActionRequest.MeasureProgressInput mp : req.getMeasureProgress()) {
                Measure m = measureMapper.selectById(mp.getMeasureId());
                if (m != null && id.equals(m.getProblemId())) {
                    if (mp.getProgress() != null) m.setProgress(mp.getProgress());
                    if (mp.getStatus() != null) m.setStatus(mp.getStatus());
                    measureMapper.updateById(m);
                }
            }
        }

        recordHistory(id, Stage.IMPLEMENT, req.getNote(), null, null);

        if (Boolean.TRUE.equals(req.getAdvance())) {
            Stage next = stageEngine.advance(p, Stage.Branch.ofNullable(p.getBranch()));
            p.setCurrentStage(next.code);
            p.setStatus(next.deriveProblemStatus(false));
            p.setProgress(92);
            p.setLatest("督导落实完成，进入评价反馈。");
        } else {
            int avg = avgProgress(id);
            p.setProgress(Math.max(p.getProgress(), 60 + avg / 3));
            p.setLatest("督导落实中，整体进度 " + avg + "%");
        }
        problemMapper.updateById(p);
        return p;
    }

    // ============================ 评价 ============================

    @Transactional
    public Problem evaluate(String id, EvaluateActionRequest req) {
        Problem p = required(id);
        stageEngine.validateStageAction(p, Stage.EVALUATE);

        UserContext.CurrentUser u = UserContext.get();
        Evaluation e = new Evaluation();
        e.setProblemId(id);
        e.setEvaluatorId(u.id());
        e.setEvaluatorName(u.name());
        e.setParty(req.getParty());
        e.setQuality(req.getQuality());
        e.setSpeed(req.getSpeed());
        e.setCollab(req.getCollab());
        e.setSatisfaction(req.getSatisfaction());
        BigDecimal overall = req.getQuality().add(req.getSpeed()).add(req.getCollab()).add(req.getSatisfaction())
                .divide(BigDecimal.valueOf(4), 1, RoundingMode.HALF_UP);
        e.setOverall(overall);
        e.setComment(req.getComment());
        e.setArchiveBestPractice(Boolean.TRUE.equals(req.getArchiveBestPractice()));
        evaluationMapper.insert(e);

        p.setStatus("done");
        p.setProgress(100);
        p.setLatest("评价完成，综合 " + overall + " 分。问题闭环。");
        problemMapper.updateById(p);

        recordHistory(id, Stage.EVALUATE, "评价完成 (" + req.getParty() + ", " + overall + "/5)", null, null);
        return p;
    }

    // ============================ 通用 ============================

    public Problem required(String id) {
        Problem p = problemMapper.selectById(id);
        if (p == null) throw new BizException("问题不存在: " + id);
        return p;
    }

    private void recordHistory(String problemId, Stage stage, String note, List<String> files, String branchChoice) {
        UserContext.CurrentUser u = UserContext.get();
        StageHistory h = new StageHistory();
        h.setProblemId(problemId);
        h.setStage(stage.code);
        h.setOccurredAt(LocalDateTime.now());
        h.setActorUserId(u.id());
        h.setActorName(u.name());
        h.setActorDept(u.dept());
        h.setNote(note);
        h.setFiles(files != null ? files : Collections.emptyList());
        h.setBranchChoice(branchChoice);
        historyMapper.insert(h);
    }

    private Long currentUid() { return UserContext.get().id(); }

    private int avgProgress(String problemId) {
        List<Measure> ms = measureMapper.selectList(new QueryWrapper<Measure>().eq("problem_id", problemId));
        if (ms.isEmpty()) return 0;
        return ms.stream().mapToInt(m -> m.getProgress() != null ? m.getProgress() : 0).sum() / ms.size();
    }
}
