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
@TableName(value = "message", autoResultMap = true)
public class Message {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private Long actorUserId;
    private String actorName;
    private String content;

    @TableField(typeHandler = JacksonTypeHandler.class)
    private List<String> mentions;

    private LocalDateTime occurredAt;

    @TableField(typeHandler = JacksonTypeHandler.class)
    private List<Long> readBy;
}
