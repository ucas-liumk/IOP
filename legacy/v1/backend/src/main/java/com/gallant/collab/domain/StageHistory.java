package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import com.baomidou.mybatisplus.extension.handlers.JacksonTypeHandler;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
@TableName(value = "stage_history", autoResultMap = true)
public class StageHistory {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private String stage;
    private LocalDateTime occurredAt;
    private Long actorUserId;
    private String actorName;
    private String actorDept;
    private String note;

    @TableField(typeHandler = JacksonTypeHandler.class)
    private List<String> files;

    private String branchChoice;
    private LocalDateTime createdAt;
}
