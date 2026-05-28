package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("measure")
public class Measure {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private String code;
    private String title;
    private String owner;
    private String status;       // proposed/drafting/approved/in_progress/completed
    private Boolean hasDispute;
    private Integer progress;
    private Integer displayOrder;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
