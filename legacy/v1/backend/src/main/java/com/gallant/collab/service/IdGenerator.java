package com.gallant.collab.service;

import com.gallant.collab.mapper.ProblemMapper;
import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import org.springframework.stereotype.Component;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;

/** 业务编号生成器。格式: WTyyyyMMdd-NNN */
@Component
public class IdGenerator {

    private static final DateTimeFormatter FMT = DateTimeFormatter.ofPattern("yyyyMMdd");
    private final ProblemMapper problemMapper;

    public IdGenerator(ProblemMapper problemMapper) {
        this.problemMapper = problemMapper;
    }

    public synchronized String nextProblemId() {
        String date = LocalDate.now().format(FMT);
        String prefix = "WT" + date + "-";
        long count = problemMapper.selectCount(new QueryWrapper<com.gallant.collab.domain.Problem>().likeRight("id", prefix));
        return prefix + String.format("%03d", count + 1);
    }
}
