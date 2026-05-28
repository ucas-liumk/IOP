package com.gallant.collab.domain;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("attachment")
public class Attachment {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String problemId;
    private String stage;
    private String fileName;
    private Long fileSize;
    private String contentType;
    private String objectKey;
    private Long uploaderId;
    private String uploaderName;
    private LocalDateTime uploadedAt;
}
