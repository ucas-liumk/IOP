package com.gallant.collab.stage;

import com.gallant.collab.common.BizException;
import com.gallant.collab.domain.Problem;
import org.springframework.stereotype.Component;

import java.util.Optional;

/** 状态机引擎: 单一职责 - 决定问题在某动作下应推进到哪个阶段, 同时校验合法性。 */
@Component
public class StageEngine {

    /**
     * 推进当前阶段到下一阶段。
     * @param problem      当前问题
     * @param branchChoice 仅在当前阶段为 PROPOSE 时使用 (DISPUTE / CONSENSUS)
     * @return 下一阶段 (若已到终态, 返回 EVALUATE)
     */
    public Stage advance(Problem problem, Stage.Branch branchChoice) {
        Stage current = Stage.of(problem.getCurrentStage());
        if ("done".equals(problem.getStatus())) {
            throw new BizException("问题已办结, 不允许再推进");
        }
        Stage.Branch branch = current == Stage.PROPOSE ? branchChoice : Stage.Branch.ofNullable(problem.getBranch());
        if (current == Stage.PROPOSE && branchChoice == null) {
            throw new BizException("研提举措阶段必须明确分支 (dispute/consensus)");
        }
        Optional<Stage> next = Stage.next(current, branch);
        if (next.isEmpty()) {
            // 已在 evaluate, 推进意味着办结
            return Stage.EVALUATE;
        }
        return next.get();
    }

    /** 校验某阶段动作合法性 (仅可在当前阶段或之前阶段执行) */
    public void validateStageAction(Problem problem, Stage requestedStage) {
        Stage current = Stage.of(problem.getCurrentStage());
        if (requestedStage.seq > current.seq) {
            throw new BizException("不能在未到达的阶段执行动作: " + requestedStage.label);
        }
        // 已办结仅允许评价阶段的查看, 不允许写
        if ("done".equals(problem.getStatus()) && requestedStage != Stage.EVALUATE) {
            throw new BizException("问题已办结, 不允许在 " + requestedStage.label + " 阶段提交");
        }
    }
}
