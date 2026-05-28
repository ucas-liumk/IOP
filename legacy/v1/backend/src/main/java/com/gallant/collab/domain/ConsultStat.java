package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDate;
import java.time.LocalDateTime;

@Data
@TableName("consult_stat")
public class ConsultStat {
    @TableId
    private String problemId;
    private Integer totalCount;
    private Integer supportCount;
    private Integer neutralCount;
    private Integer opposeCount;
    private LocalDate startDate;
    private LocalDate endDate;
    private String brief;
    private String revision;
    private LocalDateTime updatedAt;
}
