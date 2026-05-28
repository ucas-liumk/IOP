package com.gallant.collab.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.gallant.collab.domain.Problem;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;
import java.util.Map;

@Mapper
public interface ProblemMapper extends BaseMapper<Problem> {

    @Select("""
        SELECT current_stage AS k, COUNT(*) AS v
        FROM problem
        WHERE status NOT IN ('done')
        GROUP BY current_stage
        """)
    List<Map<String, Object>> countByStage();

    @Select("""
        SELECT category AS k, COUNT(*) AS v
        FROM problem
        GROUP BY category
        ORDER BY COUNT(*) DESC
        """)
    List<Map<String, Object>> countByCategory();

    @Select("""
        SELECT handler_dept AS k, COUNT(*) AS v
        FROM problem
        WHERE handler_dept IS NOT NULL AND handler_dept <> '—'
        GROUP BY handler_dept
        ORDER BY COUNT(*) DESC
        LIMIT 10
        """)
    List<Map<String, Object>> topHandlerDepts();

    @Select("""
        SELECT submitter_dept AS k, COUNT(*) AS v
        FROM problem
        GROUP BY submitter_dept
        ORDER BY COUNT(*) DESC
        LIMIT 10
        """)
    List<Map<String, Object>> topSubmitterDepts();

    @Select("""
        SELECT handler_dept AS k, COUNT(*) AS v
        FROM problem
        WHERE overdue = TRUE
        GROUP BY handler_dept
        ORDER BY COUNT(*) DESC
        LIMIT 5
        """)
    List<Map<String, Object>> overdueByDept();

    /** 6+ 个月的提报与办结趋势 */
    @Select("""
        WITH months AS (
          SELECT to_char(date_trunc('month', d), 'YYYY.MM') AS month, date_trunc('month', d) AS m
          FROM generate_series(date_trunc('month', CURRENT_DATE) - INTERVAL '11 months',
                               date_trunc('month', CURRENT_DATE),
                               INTERVAL '1 month') d
        ),
        sub AS (
          SELECT to_char(date_trunc('month', submit_date), 'YYYY.MM') AS month, COUNT(*) AS submit
          FROM problem GROUP BY 1
        ),
        dn AS (
          SELECT to_char(date_trunc('month', occurred_at), 'YYYY.MM') AS month, COUNT(DISTINCT problem_id) AS done
          FROM stage_history WHERE stage = 'evaluate' GROUP BY 1
        )
        SELECT m.month AS month, COALESCE(sub.submit, 0) AS submit, COALESCE(dn.done, 0) AS done
        FROM months m
        LEFT JOIN sub ON m.month = sub.month
        LEFT JOIN dn  ON m.month = dn.month
        ORDER BY m.m
        """)
    List<Map<String, Object>> monthlyTrend();

    @Select("""
        SELECT handler_dept AS name, ROUND(AVG(overall)::numeric, 1) AS score, COUNT(*) AS evaluations
        FROM evaluation e
        JOIN problem p ON p.id = e.problem_id
        GROUP BY handler_dept
        ORDER BY score DESC
        LIMIT 10
        """)
    List<Map<String, Object>> satisfactionRanking();
}
