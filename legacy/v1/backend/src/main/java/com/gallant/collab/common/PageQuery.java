package com.gallant.collab.common;

import lombok.Data;

@Data
public class PageQuery {
    private int page = 1;
    private int size = 20;

    public int offset() {
        return (Math.max(page, 1) - 1) * Math.max(size, 1);
    }
}
