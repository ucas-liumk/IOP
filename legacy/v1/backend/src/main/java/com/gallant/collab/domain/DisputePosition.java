package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("dispute_position")
public class DisputePosition {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long disputeId;
    private String party;
    @TableField("view_text")
    private String view;
    private LocalDateTime createdAt;
}
