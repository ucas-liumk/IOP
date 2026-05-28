package com.gallant.collab.stage;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;

/** 阶段定义与状态机的纯单元测试, 无需 Spring 上下文 */
class StageTest {

    @Test
    @DisplayName("枚举 code 与序号正确")
    void enumCodes() {
        assertThat(Stage.SUBMIT.code).isEqualTo("submit");
        assertThat(Stage.EVALUATE.seq).isEqualTo(8);
        assertThat(Stage.of("propose")).isEqualTo(Stage.PROPOSE);
    }

    @Test
    @DisplayName("公共节点路径: submit → review → propose → consult → implement → evaluate (无分支即视为共识)")
    void commonPath() {
        List<Stage> path = Stage.pathFor(Stage.Branch.CONSENSUS);
        assertThat(path).extracting(s -> s.code)
                .containsExactly("submit", "review", "propose", "consult", "implement", "evaluate");
    }

    @Test
    @DisplayName("争议路径: submit → review → propose → meeting → arbitrate → implement → evaluate")
    void disputePath() {
        List<Stage> path = Stage.pathFor(Stage.Branch.DISPUTE);
        assertThat(path).extracting(s -> s.code)
                .containsExactly("submit", "review", "propose", "meeting", "arbitrate", "implement", "evaluate");
    }

    @Test
    @DisplayName("propose + dispute → meeting")
    void proposeToMeeting() {
        assertThat(Stage.next(Stage.PROPOSE, Stage.Branch.DISPUTE)).contains(Stage.MEETING);
    }

    @Test
    @DisplayName("propose + consensus → consult")
    void proposeToConsult() {
        assertThat(Stage.next(Stage.PROPOSE, Stage.Branch.CONSENSUS)).contains(Stage.CONSULT);
    }

    @Test
    @DisplayName("arbitrate / consult 都汇聚到 implement")
    void converge() {
        assertThat(Stage.next(Stage.ARBITRATE, Stage.Branch.DISPUTE)).contains(Stage.IMPLEMENT);
        assertThat(Stage.next(Stage.CONSULT, Stage.Branch.CONSENSUS)).contains(Stage.IMPLEMENT);
    }

    @Test
    @DisplayName("evaluate 终态")
    void evaluateTerminal() {
        assertThat(Stage.next(Stage.EVALUATE, null)).isEqualTo(Optional.empty());
    }

    @Test
    @DisplayName("Branch.ofNullable 容错")
    void branchParse() {
        assertThat(Stage.Branch.ofNullable(null)).isNull();
        assertThat(Stage.Branch.ofNullable("dispute")).isEqualTo(Stage.Branch.DISPUTE);
        assertThat(Stage.Branch.ofNullable("CONSENSUS")).isEqualTo(Stage.Branch.CONSENSUS);
    }

    @Test
    @DisplayName("阶段映射状态")
    void statusMapping() {
        assertThat(Stage.SUBMIT.deriveProblemStatus(false)).isEqualTo("pending");
        assertThat(Stage.MEETING.deriveProblemStatus(false)).isEqualTo("meeting");
        assertThat(Stage.ARBITRATE.deriveProblemStatus(false)).isEqualTo("arbitrate");
        assertThat(Stage.CONSULT.deriveProblemStatus(false)).isEqualTo("consulting");
        assertThat(Stage.IMPLEMENT.deriveProblemStatus(false)).isEqualTo("processing");
        assertThat(Stage.EVALUATE.deriveProblemStatus(true)).isEqualTo("done");
    }
}
