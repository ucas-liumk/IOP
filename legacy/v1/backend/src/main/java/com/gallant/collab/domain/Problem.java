package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import com.baomidou.mybatisplus.extension.handlers.JacksonTypeHandler;
import lombok.Data;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.List;

@Data
@TableName(value = "problem", autoResultMap = true)
public class Problem {
    @TableId
    private String id;
    private String title;
    private String description;
    private String category;
    private String priority;
    private String status;
    private String branch;
    private String currentStage;
    private Long submitterId;
    private String submitterDept;
    private String handlerName;
    private String handlerDept;
    private LocalDate submitDate;
    private LocalDate dueDate;
    private Integer progress;
    private Boolean overdue;
    private Integer overdueDays;
    private String latest;

    @TableField(typeHandler = JacksonTypeHandler.class)
    private List<String> tags;

    @TableField(typeHandler = JacksonTypeHandler.class)
    private List<String> participants;

    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
