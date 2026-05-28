package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

@Data
@TableName("evaluation")
public class Evaluation {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private Long evaluatorId;
    private String evaluatorName;
    private String party;
    private BigDecimal quality;
    private BigDecimal speed;
    private BigDecimal collab;
    private BigDecimal satisfaction;
    private BigDecimal overall;
    @TableField("comment_text")
    private String comment;
    private Boolean archiveBestPractice;
    private LocalDateTime createdAt;
}
