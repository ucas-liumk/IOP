package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("dispute")
public class Dispute {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private String point;
    private String resolution;
    private Integer displayOrder;
    private LocalDateTime createdAt;
}
