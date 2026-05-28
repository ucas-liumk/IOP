package com.gallant.collab.common;

import lombok.Getter;

/** 业务异常。code != 0 表示业务失败, message 直接给前端展示 */
@Getter
public class BizException extends RuntimeException {
    private final int code;

    public BizException(String message) {
        this(40000, message);
    }

    public BizException(int code, String message) {
        super(message);
        this.code = code;
    }
}
