package com.gallant.collab.stage;

import com.gallant.collab.common.BizException;

import java.util.Arrays;
import java.util.List;
import java.util.Optional;

/**
 * 8 节点状态机定义。流程:
 * <pre>
 *  submit → review → propose ─┬─[dispute]→ meeting → arbitrate ─┐
 *                             └─[consensus]→ consult ────────────┴→ implement → evaluate
 * </pre>
 */
public enum Stage {
    SUBMIT   ("submit",    "问题提报",  null,             1),
    REVIEW   ("review",    "审核分办",  null,             2),
    PROPOSE  ("propose",   "研提举措",  null,             3),
    MEETING  ("meeting",   "会商研究",  Branch.DISPUTE,   4),
    ARBITRATE("arbitrate", "争议裁决",  Branch.DISPUTE,   5),
    CONSULT  ("consult",   "征求意见",  Branch.CONSENSUS, 6),
    IMPLEMENT("implement", "督导落实",  null,             7),
    EVALUATE ("evaluate",  "评价反馈",  null,             8);

    public final String code;
    public final String label;
    public final Branch branch;   // null = 公共; DISPUTE / CONSENSUS = 仅出现在该分支
    public final int seq;

    Stage(String code, String label, Branch branch, int seq) {
        this.code = code; this.label = label; this.branch = branch; this.seq = seq;
    }

    public enum Branch { DISPUTE, CONSENSUS;
        public static Branch ofNullable(String s) {
            if (s == null) return null;
            return switch (s.toLowerCase()) {
                case "dispute"   -> DISPUTE;
                case "consensus" -> CONSENSUS;
                default -> throw new BizException("无效的分支: " + s);
            };
        }
        public String code() { return name().toLowerCase(); }
    }

    public static Stage of(String code) {
        return Arrays.stream(values()).filter(s -> s.code.equalsIgnoreCase(code)).findFirst()
                .orElseThrow(() -> new BizException("未知阶段: " + code));
    }

    /** 该 problem.branch 下完整路径 (按顺序) */
    public static List<Stage> pathFor(Branch branch) {
        return Arrays.stream(values()).filter(s -> s.branch == null || s.branch == branch).toList();
    }

    /** 根据当前阶段 + 分支计算下一个阶段; 终态返回 empty */
    public static Optional<Stage> next(Stage current, Branch branch) {
        return switch (current) {
            case PROPOSE -> {
                if (branch == null) yield Optional.of(CONSULT); // 默认走共识
                yield Optional.of(branch == Branch.DISPUTE ? MEETING : CONSULT);
            }
            case ARBITRATE, CONSULT -> Optional.of(IMPLEMENT);
            case EVALUATE -> Optional.empty();
            default -> {
                int idx = current.ordinal();
                Branch b = branch;
                Optional<Stage> found = Optional.empty();
                for (int i = idx + 1; i < values().length; i++) {
                    Stage s = values()[i];
                    if (s.branch == null || s.branch == b) { found = Optional.of(s); break; }
                }
                yield found;
            }
        };
    }

    /** 根据当前阶段映射对外可见状态 */
    public String deriveProblemStatus(boolean evaluated) {
        if (evaluated) return "done";
        return switch (this) {
            case SUBMIT -> "pending";
            case MEETING -> "meeting";
            case ARBITRATE -> "arbitrate";
            case CONSULT -> "consulting";
            default -> "processing";
        };
    }
}
