package com.gallant.collab.service;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.gallant.collab.domain.Problem;
import com.gallant.collab.mapper.ProblemMapper;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
@RequiredArgsConstructor
public class DashboardService {

    private final ProblemMapper problemMapper;

    /** 全局态势看板数据集合 */
    public DashboardData overview() {
        DashboardData d = new DashboardData();
        d.setKpis(buildKpis());
        d.setProcessingBreakdown(problemMapper.countByStage());
        d.setCategories(problemMapper.countByCategory());
        d.setTopSubmitterDepts(problemMapper.topSubmitterDepts());
        d.setTopHandlerDepts(problemMapper.topHandlerDepts());
        d.setOverdueByDept(problemMapper.overdueByDept());
        d.setTrend(problemMapper.monthlyTrend());
        d.setSatisfaction(problemMapper.satisfactionRanking());
        d.setDisputeStats(buildDisputeStats());
        return d;
    }

    private Kpis buildKpis() {
        Kpis k = new Kpis();
        k.setTotal(problemMapper.selectCount(null));
        k.setPendingReview(countWhere("current_stage", "review"));
        k.setPendingAssign(countByStatus("pending"));
        k.setProcessing(countByStatus("processing") + countByStatus("meeting")
                + countByStatus("arbitrate") + countByStatus("consulting"));
        k.setDone(countByStatus("done"));
        k.setOverdue(countWhere("overdue", true));
        return k;
    }

    private DisputeStats buildDisputeStats() {
        DisputeStats s = new DisputeStats();
        long total = problemMapper.selectCount(new QueryWrapper<Problem>().ne("current_stage", "submit"));
        long dispute = problemMapper.selectCount(new QueryWrapper<Problem>().eq("branch", "dispute"));
        long consensus = problemMapper.selectCount(new QueryWrapper<Problem>().eq("branch", "consensus"));
        long arbitrated = problemMapper.selectCount(new QueryWrapper<Problem>().eq("branch", "dispute")
                .in("current_stage", List.of("implement", "evaluate")));
        s.setTotalPropose(total);
        s.setWithDispute(dispute);
        s.setConsultPath(consensus);
        s.setArbitrateDone(arbitrated);
        s.setDisputeRate(total == 0 ? 0 : Math.round((double) dispute / total * 1000) / 10.0);
        s.setAvgMeetings(2.4); // 简化: 实际可基于 stage_history 计算
        return s;
    }

    private long countByStatus(String status) {
        return problemMapper.selectCount(new QueryWrapper<Problem>().eq("status", status));
    }

    private long countWhere(String col, Object v) {
        return problemMapper.selectCount(new QueryWrapper<Problem>().eq(col, v));
    }

    @Data
    public static class DashboardData {
        private Kpis kpis;
        private List<Map<String, Object>> processingBreakdown;
        private List<Map<String, Object>> categories;
        private List<Map<String, Object>> topSubmitterDepts;
        private List<Map<String, Object>> topHandlerDepts;
        private List<Map<String, Object>> overdueByDept;
        private List<Map<String, Object>> trend;
        private List<Map<String, Object>> satisfaction;
        private DisputeStats disputeStats;
    }

    @Data
    public static class Kpis {
        private long total;
        private long pendingReview;
        private long pendingAssign;
        private long processing;
        private long done;
        private long overdue;
    }

    @Data
    public static class DisputeStats {
        private long totalPropose;
        private long withDispute;
        private long consultPath;
        private long arbitrateDone;
        private double disputeRate;
        private double avgMeetings;
    }
}
