package com.gallant.collab.service;

import com.gallant.collab.AbstractIntegrationTest;
import com.gallant.collab.common.PageQuery;
import com.gallant.collab.common.PageResult;
import com.gallant.collab.domain.Problem;
import com.gallant.collab.dto.ProblemDtos;
import com.gallant.collab.dto.ProblemDtos.*;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

import static org.assertj.core.api.Assertions.*;

class ProblemServiceTest extends AbstractIntegrationTest {

    @Autowired ProblemService service;

    @Test
    @DisplayName("分页查询全部 - 共 8 个种子问题")
    void pageAll() {
        PageResult<Problem> r = service.page(new PageQuery(), null, null, null, null, null);
        assertThat(r.getTotal()).isEqualTo(8);
        assertThat(r.getItems()).hasSize(8);
    }

    @Test
    @DisplayName("按状态过滤 - meeting 应只返回 1 个")
    void filterMeeting() {
        PageResult<Problem> r = service.page(new PageQuery(), "meeting", null, null, null, null);
        assertThat(r.getItems()).hasSize(1);
        assertThat(r.getItems().get(0).getId()).isEqualTo("WT20260510-001");
    }

    @Test
    @DisplayName("详情包含 history, measures, disputes")
    void detail() {
        ProblemDetail d = service.detail("WT20260510-001");
        assertThat(d.getProblem().getTitle()).contains("H2 路线图");
        assertThat(d.getHistory()).hasSize(5);
        assertThat(d.getMeasures()).hasSize(2);
        assertThat(d.getDisputes()).hasSize(2);
    }

    @Test
    @DisplayName("创建问题 - 生成业务 id, 写入 submit 历史")
    void createProblem() {
        CreateProblemRequest req = new CreateProblemRequest();
        req.setTitle("自动化测试问题");
        req.setDescription("desc");
        req.setCategory("运营效率");
        req.setPriority("high");
        req.setDueDate(LocalDate.now().plusDays(7));
        req.setTags(List.of("测试"));
        req.setParticipants(List.of("CTO 办公室"));

        Problem p = service.create(req);
        assertThat(p.getId()).matches("WT\\d{8}-\\d{3}");
        // 创建后立即等待审核
        assertThat(p.getCurrentStage()).isEqualTo("review");
        assertThat(p.getStatus()).isEqualTo("pending");

        ProblemDetail d = service.detail(p.getId());
        assertThat(d.getHistory()).hasSize(1);
        assertThat(d.getHistory().get(0).getStage()).isEqualTo("submit");
    }

    @Test
    @DisplayName("审核通过 - 推进到 propose, 状态变 processing")
    void reviewApprove() {
        // 用待审核的问题 #7
        ReviewActionRequest req = new ReviewActionRequest();
        req.setDecision("approve");
        req.setReviewNote("数据分级标准重要, 同意立项");
        req.setHandlerName("数据中台");
        req.setHandlerDept("数据中台");
        req.setPriority("high");
        req.setDueDate(LocalDate.now().plusDays(20));
        req.setAssignNote("由数据中台牵头, 业务部门配合");

        Problem p = service.review("WT20260516-007", req);
        assertThat(p.getCurrentStage()).isEqualTo("propose");
        assertThat(p.getStatus()).isEqualTo("processing");
        assertThat(p.getHandlerDept()).isEqualTo("数据中台");

        ProblemDetail d = service.detail("WT20260516-007");
        assertThat(d.getHistory()).hasSize(2);
        assertThat(d.getHistory().get(1).getStage()).isEqualTo("review");
    }

    @Test
    @DisplayName("研提举措 - 标记争议 → 进入 meeting, 状态变 meeting")
    void proposeWithDispute() {
        // 用 propose 阶段且无 branch 的问题 #2
        ProposeActionRequest req = new ProposeActionRequest();
        req.setHasDispute(true);
        req.setNote("研提 2 项举措, 第 1 项有争议");

        ProblemDtos.ProposeActionRequest.MeasureInput m1 = new ProblemDtos.ProposeActionRequest.MeasureInput();
        m1.setCode("M1"); m1.setTitle("扩容客服"); m1.setOwner("客户服务部"); m1.setHasDispute(true);
        ProblemDtos.ProposeActionRequest.MeasureInput m2 = new ProblemDtos.ProposeActionRequest.MeasureInput();
        m2.setCode("M2"); m2.setTitle("SLA 周会"); m2.setOwner("运营优化办"); m2.setHasDispute(false);
        req.setMeasures(List.of(m1, m2));

        ProblemDtos.ProposeActionRequest.DisputeInput d = new ProblemDtos.ProposeActionRequest.DisputeInput();
        d.setPoint("是否需要追加预算");
        ProblemDtos.ProposeActionRequest.DisputePositionInput pos1 = new ProblemDtos.ProposeActionRequest.DisputePositionInput();
        pos1.setParty("客户服务部"); pos1.setView("必须追加");
        ProblemDtos.ProposeActionRequest.DisputePositionInput pos2 = new ProblemDtos.ProposeActionRequest.DisputePositionInput();
        pos2.setParty("财务部"); pos2.setView("应内部消化");
        d.setPositions(List.of(pos1, pos2));
        req.setDisputes(List.of(d));

        Problem p = service.propose("WT20260512-002", req);
        assertThat(p.getCurrentStage()).isEqualTo("meeting");
        assertThat(p.getBranch()).isEqualTo("dispute");
        assertThat(p.getStatus()).isEqualTo("meeting");

        ProblemDetail det = service.detail("WT20260512-002");
        assertThat(det.getMeasures()).hasSize(2);
        assertThat(det.getDisputes()).hasSize(1);
        assertThat(det.getDisputes().get(0).getPositions()).hasSize(2);
    }

    @Test
    @DisplayName("研提举措 - 无争议 → 进入 consult, 状态变 consulting")
    void proposeNoDispute() {
        ProposeActionRequest req = new ProposeActionRequest();
        req.setHasDispute(false);
        req.setNote("3 项举措一致通过, 转征求意见");
        ProblemDtos.ProposeActionRequest.MeasureInput m = new ProblemDtos.ProposeActionRequest.MeasureInput();
        m.setCode("M1"); m.setTitle("增设晚高峰排班"); m.setOwner("客户服务部"); m.setHasDispute(false);
        req.setMeasures(List.of(m));

        Problem p = service.propose("WT20260512-002", req);
        assertThat(p.getCurrentStage()).isEqualTo("consult");
        assertThat(p.getBranch()).isEqualTo("consensus");
        assertThat(p.getStatus()).isEqualTo("consulting");
    }

    @Test
    @DisplayName("完整争议流程: review → propose(dispute) → meeting → arbitrate → implement → evaluate → done")
    void fullDisputeFlow() {
        // 从问题 #7 (pending, 仅有 submit 历史) 起始
        String id = "WT20260516-007";

        ReviewActionRequest review = new ReviewActionRequest();
        review.setDecision("approve");
        review.setReviewNote("通过");
        review.setHandlerName("数据中台"); review.setHandlerDept("数据中台");
        service.review(id, review);

        ProposeActionRequest propose = new ProposeActionRequest();
        propose.setHasDispute(true);
        propose.setNote("两套方案存在分歧");
        ProblemDtos.ProposeActionRequest.MeasureInput m = new ProblemDtos.ProposeActionRequest.MeasureInput();
        m.setCode("M1"); m.setTitle("方案A"); m.setOwner("数据中台"); m.setHasDispute(true);
        propose.setMeasures(List.of(m));
        ProblemDtos.ProposeActionRequest.DisputeInput d = new ProblemDtos.ProposeActionRequest.DisputeInput();
        d.setPoint("分级粒度");
        ProblemDtos.ProposeActionRequest.DisputePositionInput pos = new ProblemDtos.ProposeActionRequest.DisputePositionInput();
        pos.setParty("数据中台"); pos.setView("4 级");
        d.setPositions(List.of(pos));
        propose.setDisputes(List.of(d));
        service.propose(id, propose);

        MeetingActionRequest mtg = new MeetingActionRequest();
        mtg.setSummary("达成 3 项共识, 1 项遗留"); mtg.setNote("会商完成");
        mtg.setAdvance(true);
        service.meeting(id, mtg);

        ArbitrateActionRequest arb = new ArbitrateActionRequest();
        arb.setArbitrator("CTO 办公室");
        arb.setDate(LocalDate.now());
        arb.setOverall("批准方案 A 试点 1 个季度");
        arb.setNote("发布裁决");
        service.arbitrate(id, arb);

        ImplementActionRequest imp = new ImplementActionRequest();
        imp.setNote("阶段性完成"); imp.setAdvance(true);
        service.implement(id, imp);

        EvaluateActionRequest ev = new EvaluateActionRequest();
        ev.setParty("提报方");
        ev.setQuality(BigDecimal.valueOf(4.5));
        ev.setSpeed(BigDecimal.valueOf(4.0));
        ev.setCollab(BigDecimal.valueOf(4.5));
        ev.setSatisfaction(BigDecimal.valueOf(4.6));
        ev.setComment("整体流程高效");
        service.evaluate(id, ev);

        Problem finalP = service.detail(id).getProblem();
        assertThat(finalP.getStatus()).isEqualTo("done");
        assertThat(finalP.getBranch()).isEqualTo("dispute");
        ProblemDetail det = service.detail(id);
        // 1 submit + 1 review + 1 propose + 1 meeting + 1 arbitrate + 1 implement + 1 evaluate = 7
        assertThat(det.getHistory()).hasSize(7);
        assertThat(det.getEvaluations()).hasSize(1);
        assertThat(det.getEvaluations().get(0).getOverall().doubleValue()).isCloseTo(4.4, within(0.1));
    }

    @Test
    @DisplayName("已办结的问题不允许再推进")
    void cannotAdvanceDone() {
        ProposeActionRequest req = new ProposeActionRequest();
        req.setHasDispute(false);
        req.setNote("尝试推进");
        assertThatThrownBy(() -> service.propose("WT20260420-004", req))
                .hasMessageContaining("已办结");
    }
}
